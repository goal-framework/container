package container

import (
	"github.com/goal-web/contracts"
)

type FuncTypeException struct {
	contracts.Exception

	Fn any
}

type DIKindException struct {
	contracts.Exception

	Object any
}

type CanNotInjectException struct {
	contracts.Exception

	Arg string
}

type DIFieldException struct {
	contracts.Exception

	Object any
	Field  string
	Target any
}
