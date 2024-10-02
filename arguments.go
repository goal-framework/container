package container

import (
	"github.com/goal-web/supports/utils"
	"reflect"
	"runtime"
)

type ArgumentsTypeMap map[string][]any

func NewArgumentsTypeMap(args []any) ArgumentsTypeMap {
	argsTypeMap := ArgumentsTypeMap{}
	for _, arg := range args {
		argTypeKey := utils.GetTypeKey(reflect.TypeOf(arg))
		if argTypeKey == "" {
			argTypeKey = runtime.FuncForPC(reflect.ValueOf(arg).Pointer()).Name()
		}
		argsTypeMap[argTypeKey] = append(argsTypeMap[argTypeKey], arg)
	}
	return argsTypeMap
}

func (args ArgumentsTypeMap) Pull(key string) (arg any) {
	if item, exits := args[key]; exits && len(item) >= 1 {
		arg = item[0]
		args[key] = item[1:]
		return
	}
	return nil
}

// FindConvertibleArg 找到可转换的参数
func (args ArgumentsTypeMap) FindConvertibleArg(targetKey string, targetType reflect.Type) any {
	for key, items := range args {
		for _, arg := range items {
			if reflect.TypeOf(arg).ConvertibleTo(targetType) {
				if key != targetKey {
					return reflect.ValueOf(arg).Convert(targetType).Interface()
				}
				return arg
			}
		}
	}
	return nil
}
