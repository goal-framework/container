package container

import (
	"fmt"
	"github.com/goal-web/contracts"
	"github.com/goal-web/supports/exceptions"
	"github.com/goal-web/supports/logs"
	"github.com/goal-web/supports/utils"
	"reflect"
	"sync"
)

var (
	CallerTypeError    = exceptions.New("The argument type must be a function with a return value")
	StructPtrTypeError = exceptions.New("The parameter must be a pointer to a structure")
)

type Container struct {
	binds        map[string]contracts.MagicalFunc
	singletons   map[string]contracts.MagicalFunc
	instances    sync.Map
	aliases      sync.Map
	argProviders []func(key string, p reflect.Type, arguments ArgumentsTypeMap) any
}

func newInstanceProvider(provider any) contracts.MagicalFunc {
	magicalFn := NewMagicalFunc(provider)
	if magicalFn.NumOut() != 1 {
		exceptions.Throw(CallerTypeError)
	}
	return magicalFn
}

func New() contracts.Container {
	container := &Container{}
	container.argProviders = []func(key string, p reflect.Type, arguments ArgumentsTypeMap) any{
		func(key string, _ reflect.Type, arguments ArgumentsTypeMap) any {
			return arguments.Pull(key) // 外部参数里面类型完全相等的参数
		},
		func(key string, _ reflect.Type, arguments ArgumentsTypeMap) any {
			return arguments.Pull(key) // 外部参数里面类型完全相等的参数
		},
		func(key string, argType reflect.Type, arguments ArgumentsTypeMap) any {
			return arguments.FindConvertibleArg(key, argType) // 外部参数可转换的参数
		},
		func(key string, argType reflect.Type, arguments ArgumentsTypeMap) any {
			return container.GetByArguments(key, arguments) // 从容器中获取参数
		},
		func(key string, argType reflect.Type, arguments ArgumentsTypeMap) any {
			defer func() {
				if err := recover(); err != nil {
					logs.Default().WithField("err", err).Error("Unable to inject parameter " + key)
				}
			}()
			// 尝试 new 一个然后通过容器注入
			var (
				tempInstance any
				isPtr        = argType.Kind() == reflect.Ptr
			)
			if isPtr {
				tempInstance = reflect.New(argType.Elem()).Interface()
			} else {
				tempInstance = reflect.New(argType).Interface()
			}
			container.DIByArguments(tempInstance, arguments)
			if isPtr {
				return tempInstance
			}
			return reflect.ValueOf(tempInstance).Elem().Interface()
		},
	}
	container.Flush()
	return container
}

func (container *Container) Bind(key string, provider any) {
	magicalFn := newInstanceProvider(provider)
	container.binds[container.GetKey(key)] = magicalFn
	container.Alias(key, utils.GetTypeKey(magicalFn.Returns()[0]))
}

func (container *Container) Instance(key string, instance any) {
	container.instances.Store(container.GetKey(key), instance)
}

func (container *Container) Singleton(key string, provider any) {
	magicalFn := newInstanceProvider(provider)
	container.singletons[container.GetKey(key)] = magicalFn
	container.Alias(key, utils.GetTypeKey(magicalFn.Returns()[0]))
}

func (container *Container) HasBound(key string) bool {
	key = container.GetKey(key)
	if _, existsBind := container.binds[key]; existsBind {
		return true
	}
	if _, existsSingleton := container.singletons[key]; existsSingleton {
		return true
	}
	if _, existsInstance := container.instances.Load(key); existsInstance {
		return true
	}
	return false
}

func (container *Container) Alias(key string, alias string) {
	container.aliases.Store(alias, key)
}

func (container *Container) GetKey(alias string) string {
	if value, existsAlias := container.aliases.Load(alias); existsAlias {
		return value.(string)
	}
	return alias
}

func (container *Container) Flush() {
	container.instances = sync.Map{}
	container.singletons = make(map[string]contracts.MagicalFunc)
	container.binds = make(map[string]contracts.MagicalFunc)
	container.aliases = sync.Map{}
}

func (container *Container) Get(key string, args ...any) any {
	key = container.GetKey(key)
	if tempInstance, existsInstance := container.instances.Load(key); existsInstance {
		return tempInstance
	}
	if singletonProvider, existsProvider := container.singletons[key]; existsProvider {
		value := container.Call(singletonProvider, args...)[0]
		container.instances.Store(key, value)
		return value
	}
	if instanceProvider, existsProvider := container.binds[key]; existsProvider {
		return container.Call(instanceProvider, args...)[0]
	}
	return nil
}

func (container *Container) GetByArguments(key string, arguments ArgumentsTypeMap) any {
	key = container.GetKey(key)
	if tempInstance, existsInstance := container.instances.Load(key); existsInstance {
		return tempInstance
	}
	if singletonProvider, existsProvider := container.singletons[key]; existsProvider {
		value := container.StaticCallByArguments(singletonProvider, arguments)[0]
		container.instances.Store(key, value)
		return value
	}
	if instanceProvider, existsProvider := container.binds[key]; existsProvider {
		return container.StaticCallByArguments(instanceProvider, arguments)[0]
	}
	return nil
}

// StaticCall 静态调用，直接传静态化的方法
func (container *Container) StaticCall(magicalFn contracts.MagicalFunc, args ...any) []any {
	return container.StaticCallByArguments(magicalFn, NewArgumentsTypeMap(append(args, container)))
}

// StaticCallByArguments 静态调用，直接传静态化的方法和处理好的参数
func (container *Container) StaticCallByArguments(magicalFn contracts.MagicalFunc, arguments ArgumentsTypeMap) []any {
	fnArgs := make([]reflect.Value, 0)

	for i, arg := range magicalFn.Arguments() {
		if magicalFn.IsVariadic() && i == len(magicalFn.Arguments())-1 { // 注入可变参数
			key := utils.GetTypeKey(arg.Elem())
			for _, value := range arguments[key] {
				fnArgs = append(fnArgs, reflect.ValueOf(value))
			}
		} else {
			key := utils.GetTypeKey(arg)
			fnArgs = append(fnArgs, reflect.ValueOf(container.findArg(key, arg, arguments)))
		}
	}

	results := make([]any, 0)

	for _, result := range magicalFn.Call(fnArgs) {
		results = append(results, result.Interface())
	}

	return results
}

func (container *Container) Call(fn any, args ...any) []any {
	if magicalFn, isMagicalFunc := fn.(contracts.MagicalFunc); isMagicalFunc {
		return container.StaticCall(magicalFn, args...)
	}
	return container.StaticCall(NewMagicalFunc(fn), args...)
}

func (container *Container) findArg(key string, p reflect.Type, arguments ArgumentsTypeMap) (result any) {
	for _, provider := range container.argProviders {
		if value := provider(key, p, arguments); value != nil {
			return value
		}
	}
	return
}

func (container *Container) DIByArguments(object any, arguments ArgumentsTypeMap) {
	if component, ok := object.(contracts.Component); ok {
		component.Construct(container)
		return
	}

	objectValue := reflect.ValueOf(object)

	switch objectValue.Kind() {
	case reflect.Ptr:
		if objectValue.Elem().Kind() != reflect.Struct {
			exceptions.Throw(DIKindException{
				Exception: StructPtrTypeError,
				Object:    object,
			})
		}
		objectValue = objectValue.Elem()
	default:
		exceptions.Throw(DIKindException{
			Exception: StructPtrTypeError,
			Object:    object,
		})
	}

	valueType := objectValue.Type()

	var (
		fieldNum  = objectValue.NumField()
		tempValue = reflect.New(valueType).Elem()
	)

	tempValue.Set(objectValue)

	// 遍历所有字段
	for i := 0; i < fieldNum; i++ {
		var (
			field          = valueType.Field(i)
			key            = utils.GetTypeKey(field.Type)
			fieldTags      = utils.ParseStructTag(field.Tag)
			fieldValue     = tempValue.Field(i)
			fieldInterface any
		)

		if di, existsDiTag := fieldTags["di"]; existsDiTag { // 配置了 fieldTags tag，优先用 tag 的配置
			if len(di) > 0 { // 如果指定某 di 值，优先取这个值
				fieldInterface = container.Get(di[0])
			}
			if fieldInterface == nil {
				fieldInterface = container.findArg(key, field.Type, arguments)
			}
		}

		if fieldInterface != nil {
			fieldType := reflect.TypeOf(fieldInterface)
			if fieldType.ConvertibleTo(field.Type) { // 可转换的类型
				value := reflect.ValueOf(fieldInterface)
				if key != utils.GetTypeKey(fieldType) { // 如果不是同一种类型，就转换一下
					value = value.Convert(field.Type)
				}
				fieldValue.Set(value)
			} else {
				exceptions.Throw(
					DIFieldException{
						Exception: exceptions.WithError(fmt.Errorf("it is not possible to inject %s because of a type inconsistency, where the target type is %s and the type that will be injected is %s", field.Name, field.Type.String(), fieldType.String())),
						Object:    object,
						Field:     field.Name,
						Target:    fieldType,
					},
				)
			}
		}
	}

	objectValue.Set(tempValue)
}

func (container *Container) DI(object any, args ...any) {
	container.DIByArguments(object, NewArgumentsTypeMap(append(args, container)))
}
