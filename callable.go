package container

import (
	"github.com/goal-web/contracts"
	"github.com/goal-web/supports/exceptions"
	"reflect"
	"runtime"
)

var (
	FuncTypeError = exceptions.New("The argument must be a function")
)

type magicalFunc struct {
	in         int
	out        int
	value      reflect.Value
	arguments  []reflect.Type
	returns    []reflect.Type
	isVariadic bool
}

func NewMagicalFunc(fn any) contracts.MagicalFunc {
	var (
		argValue = reflect.ValueOf(fn)
		argType  = reflect.TypeOf(fn)
	)

	if argValue.Kind() != reflect.Func {
		panic(FuncTypeException{
			Exception: FuncTypeError,
			Fn:        fn,
		})
	}

	var (
		arguments    = make([]reflect.Type, 0)
		returns      = make([]reflect.Type, 0)
		argumentsLen = argType.NumIn()
		returnsLen   = argType.NumOut()
	)

	for argIndex := 0; argIndex < argumentsLen; argIndex++ {
		arguments = append(arguments, argType.In(argIndex))
	}

	for outIndex := 0; outIndex < returnsLen; outIndex++ {
		returns = append(returns, argType.Out(outIndex))
	}

	return &magicalFunc{
		isVariadic: argType.IsVariadic(),
		in:         argumentsLen,
		out:        returnsLen,
		value:      argValue,
		arguments:  arguments,
		returns:    returns,
	}
}

func (fn *magicalFunc) Call(in []reflect.Value) []reflect.Value {
	return fn.value.Call(in)
}

func (fn *magicalFunc) Arguments() []reflect.Type {
	return fn.arguments
}

func (fn *magicalFunc) Returns() []reflect.Type {
	return fn.returns
}

func (fn *magicalFunc) NumOut() int {
	return fn.out
}

func (fn *magicalFunc) NumIn() int {
	return fn.in
}

func (fn *magicalFunc) IsVariadic() bool {
	return fn.isVariadic
}

func (fn *magicalFunc) Signature() string {
	return runtime.FuncForPC(fn.value.Pointer()).Name()
}
