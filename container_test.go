package container

import (
	"github.com/goal-framework/utils"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type DemoParam struct {
	Id string
}

func TestArgumentsTypeMap(t *testing.T) {
	args := NewArgumentsTypeMap([]interface{}{"啦啦啦", DemoParam{Id: "111"}})
	str := args.Pull("string")
	assert.True(t, str == "啦啦啦")

	args = NewArgumentsTypeMap([]interface{}{})
	assert.True(t, args.Pull("string") == nil)
}

func TestBaseContainer(t *testing.T) {
	app := New()

	app.Instance("a", "a")
	assert.True(t, app.HasBound("a"))
	assert.True(t, app.Get("a") == "a")

	app.Alias("a", "A")

	assert.True(t, app.Get("A") == "a")
	assert.True(t, app.HasBound("A"))

	app.Provide(func() DemoParam {
		return DemoParam{Id: "测试一下"}
	})

	assert.True(t, app.Get(utils.GetTypeKey(reflect.TypeOf(DemoParam{}))).(DemoParam).Id == "测试一下")

	app.Call(func(param DemoParam) {
		assert.True(t, param.Id == "测试一下")
	})

}

func TestContainerCall(t *testing.T) {
	app := New()

	app.Provide(func() DemoParam {
		return DemoParam{Id: "没有外部参数的话，从容器中获取"}
	})

	fn := func(param DemoParam) string {
		return param.Id
	}

	// 自己传参
	assert.True(t, app.Call(fn, DemoParam{Id: "优先使用外部参数"})[0] == "优先使用外部参数")

	// 不传参，使用容器中的实例
	assert.True(t, app.Call(fn)[0] == "没有外部参数的话，从容器中获取")

}

type DemoStruct struct {
	Param  DemoParam `di:""`       // 注入对应类型的实例
	Config string    `di:"config"` // 注入指定 key 的实例
}

func TestContainerDI(t *testing.T) {
	app := New()

	app.Instance("config", "通过容器设置的配置")

	app.Provide(func() DemoParam {
		return DemoParam{Id: "dd"}
	})

	demo := &DemoStruct{}

	app.DI(demo)


	assert.True(t, demo.Param.Id == "dd")
}