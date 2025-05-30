package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Shape interface {
	SetArea(int)
	GetArea() int
}

type Circle struct {
	a int
}

func (c *Circle) SetArea(a int) {
	c.a = a
}

func (c Circle) GetArea() int {
	return c.a
}

type Database interface {
	Connect() bool
}

type MySQL struct{}

func (m MySQL) Connect() bool {
	return true
}

var instance = NewContainer()

func TestContainer_Singleton(t *testing.T) {
	err := instance.Singleton(func() Shape {
		return &Circle{a: 13}
	})
	assert.NoError(t, err)

	err = instance.Call(func(s1 Shape) {
		s1.SetArea(666)
	})
	assert.NoError(t, err)

	err = instance.Call(func(s2 Shape) {
		a := s2.GetArea()
		assert.Equal(t, a, 666)
	})
	assert.NoError(t, err)
}

func TestContainer_SingletonLazy(t *testing.T) {
	err := instance.SingletonLazy(func() Shape {
		return &Circle{a: 13}
	})
	assert.NoError(t, err)

	err = instance.Call(func(s1 Shape) {
		s1.SetArea(666)
	})
	assert.NoError(t, err)

	err = instance.Call(func(s2 Shape) {
		a := s2.GetArea()
		assert.Equal(t, a, 666)
	})
	assert.NoError(t, err)
}

func TestContainer_Singleton_With_Missing_Dependency_Resolve(t *testing.T) {
	err := instance.Singleton(func(db Database) Shape {
		return &Circle{a: 13}
	})
	assert.EqualError(t, err, "container: no concrete found for: service.Database")
}

func TestContainer_Singleton_With_Resolve_That_Returns_Nothing(t *testing.T) {
	err := instance.Singleton(func() {})
	assert.Error(t, err, "container: resolver function signature is invalid")
}

func TestContainer_SingletonLazy_With_Resolve_That_Returns_Nothing(t *testing.T) {
	err := instance.SingletonLazy(func() {})
	assert.Error(t, err, "container: resolver function signature is invalid")
}

func TestContainer_Singleton_With_Resolve_That_Returns_Error(t *testing.T) {
	err := instance.Singleton(func() (Shape, error) {
		return nil, errors.New("app: error")
	})
	assert.Error(t, err, "app: error")
}

func TestContainer_SingletonLazy_With_Resolve_That_Returns_Error(t *testing.T) {
	err := instance.SingletonLazy(func() (Shape, error) {
		return nil, errors.New("app: error")
	})
	assert.NoError(t, err)

	var s Shape
	err = instance.Resolve(&s)
	assert.Error(t, err, "app: error")
}

func TestContainer_Singleton_With_NonFunction_Resolver_It_Should_Fail(t *testing.T) {
	err := instance.Singleton("STRING!")
	assert.EqualError(t, err, "container: the resolver must be a function")
}

func TestContainer_SingletonLazy_With_NonFunction_Resolver_It_Should_Fail(t *testing.T) {
	err := instance.SingletonLazy("STRING!")
	assert.EqualError(t, err, "container: the resolver must be a function")
}

func TestContainer_Singleton_With_Resolvable_Arguments(t *testing.T) {
	err := instance.Singleton(func() Shape {
		return &Circle{a: 666}
	})
	assert.NoError(t, err)

	err = instance.Singleton(func(s Shape) Database {
		assert.Equal(t, s.GetArea(), 666)
		return &MySQL{}
	})
	assert.NoError(t, err)
}

func TestContainer_SingletonLazy_With_Resolvable_Arguments(t *testing.T) {
	err := instance.SingletonLazy(func() Shape {
		return &Circle{a: 666}
	})
	assert.NoError(t, err)

	err = instance.SingletonLazy(func(s Shape) Database {
		assert.Equal(t, s.GetArea(), 666)
		return &MySQL{}
	})
	assert.NoError(t, err)

	var s Shape
	err = instance.Resolve(&s)
	assert.NoError(t, err)
}

func TestContainer_Singleton_With_Non_Resolvable_Arguments(t *testing.T) {
	instance.Reset()

	err := instance.Singleton(func(s Shape) Shape {
		return &Circle{a: s.GetArea()}
	})
	assert.EqualError(t, err, "container: resolver function signature is invalid - depends on abstract it returns")
}

func TestContainer_SingletonLazy_With_Non_Resolvable_Arguments(t *testing.T) {
	instance.Reset()

	err := instance.SingletonLazy(func(s Shape) Shape {
		return &Circle{a: s.GetArea()}
	})
	assert.EqualError(t, err, "container: resolver function signature is invalid - depends on abstract it returns")
}

func TestContainer_NamedSingleton(t *testing.T) {
	err := instance.NamedSingleton("theCircle", func() Shape {
		return &Circle{a: 13}
	})
	assert.NoError(t, err)

	var sh Shape
	err = instance.NamedResolve(&sh, "theCircle")
	assert.NoError(t, err)
	assert.Equal(t, sh.GetArea(), 13)
}

func TestContainer_NamedSingletonLazy(t *testing.T) {
	err := instance.NamedSingletonLazy("theCircle", func() Shape {
		return &Circle{a: 13}
	})
	assert.NoError(t, err)

	var sh Shape
	err = instance.NamedResolve(&sh, "theCircle")
	assert.NoError(t, err)
	assert.Equal(t, sh.GetArea(), 13)
}

func TestContainer_Transient(t *testing.T) {
	err := instance.Bind(func() Shape {
		return &Circle{a: 666}
	})
	assert.NoError(t, err)

	err = instance.Call(func(s1 Shape) {
		s1.SetArea(13)
	})
	assert.NoError(t, err)

	err = instance.Call(func(s2 Shape) {
		a := s2.GetArea()
		assert.Equal(t, a, 666)
	})
	assert.NoError(t, err)
}

func TestContainer_Transient_With_Resolve_That_Returns_Nothing(t *testing.T) {
	err := instance.Bind(func() {})
	assert.Error(t, err, "container: resolver function signature is invalid")
}

func TestContainer_Transient_With_Resolve_That_Returns_Error(t *testing.T) {
	err := instance.Bind(func() (Shape, error) {
		return nil, errors.New("app: error")
	})
	assert.Error(t, err, "app: error")

	firstCall := true
	err = instance.Bind(func() (Database, error) {
		if firstCall {
			firstCall = false
			return &MySQL{}, nil
		}
		return nil, errors.New("app: second call error")
	})
	assert.NoError(t, err)

	var db Database
	err = instance.Resolve(&db)
	assert.Error(t, err, "app: second call error")
}

func TestContainer_Transient_With_Resolve_With_Invalid_Signature_It_Should_Fail(t *testing.T) {
	err := instance.Bind(func() (Shape, Database, error) {
		return nil, nil, nil
	})
	assert.Error(t, err, "container: resolver function signature is invalid")
}

func TestContainer_NamedTransient(t *testing.T) {
	err := instance.NamedBind("theCircle", func() Shape {
		return &Circle{a: 13}
	})
	assert.NoError(t, err)

	var sh Shape
	err = instance.NamedResolve(&sh, "theCircle")
	assert.NoError(t, err)
	assert.Equal(t, sh.GetArea(), 13)
}

func TestContainer_Call_With_Multiple_Resolving(t *testing.T) {
	err := instance.Singleton(func() Shape {
		return &Circle{a: 5}
	})
	assert.NoError(t, err)

	err = instance.Singleton(func() Database {
		return &MySQL{}
	})
	assert.NoError(t, err)

	err = instance.Call(func(s Shape, m Database) {
		if _, ok := s.(*Circle); !ok {
			t.Error("Expected Circle")
		}

		if _, ok := m.(*MySQL); !ok {
			t.Error("Expected MySQL")
		}
	})
	assert.NoError(t, err)
}

func TestContainer_Call_With_Dependency_Missing_In_Chain(t *testing.T) {
	var instance = NewContainer()
	err := instance.SingletonLazy(func() (Database, error) {
		var s Shape
		if err := instance.Resolve(&s); err != nil {
			return nil, err
		}
		return &MySQL{}, nil
	})
	assert.NoError(t, err)

	err = instance.Call(func(m Database) {
		if _, ok := m.(*MySQL); !ok {
			t.Error("Expected MySQL")
		}
	})
	assert.EqualError(t, err, "container: no concrete found for: service.Shape")
}

func TestContainer_Call_With_Unsupported_Receiver_It_Should_Fail(t *testing.T) {
	err := instance.Call("STRING!")
	assert.EqualError(t, err, "container: invalid function")
}

func TestContainer_Call_With_Second_UnBounded_Argument(t *testing.T) {
	instance.Reset()

	err := instance.Singleton(func() Shape {
		return &Circle{}
	})
	assert.NoError(t, err)

	err = instance.Call(func(s Shape, d Database) {})
	assert.EqualError(t, err, "container: no concrete found for: service.Database")
}

func TestContainer_Call_With_A_Returning_Error(t *testing.T) {
	instance.Reset()

	err := instance.Singleton(func() Shape {
		return &Circle{}
	})
	assert.NoError(t, err)

	err = instance.Call(func(s Shape) error {
		return errors.New("app: some context error")
	})
	assert.EqualError(t, err, "app: some context error")
}

func TestContainer_Call_With_A_Returning_Nil_Error(t *testing.T) {
	instance.Reset()

	err := instance.Singleton(func() Shape {
		return &Circle{}
	})
	assert.NoError(t, err)

	err = instance.Call(func(s Shape) error {
		return nil
	})
	assert.Nil(t, err)
}

func TestContainer_Call_With_Invalid_Signature(t *testing.T) {
	instance.Reset()

	err := instance.Singleton(func() Shape {
		return &Circle{}
	})
	assert.NoError(t, err)

	err = instance.Call(func(s Shape) (int, error) {
		return 13, errors.New("app: some context error")
	})
	assert.EqualError(t, err, "container: receiver function signature is invalid")
}

func TestContainer_Resolve_With_Reference_As_Resolver(t *testing.T) {
	err := instance.Singleton(func() Shape {
		return &Circle{a: 5}
	})
	assert.NoError(t, err)

	err = instance.Singleton(func() Database {
		return &MySQL{}
	})
	assert.NoError(t, err)

	var (
		s Shape
		d Database
	)

	err = instance.Resolve(&s)
	assert.NoError(t, err)
	if _, ok := s.(*Circle); !ok {
		t.Error("Expected Circle")
	}

	err = instance.Resolve(&d)
	assert.NoError(t, err)
	if _, ok := d.(*MySQL); !ok {
		t.Error("Expected MySQL")
	}
}

func TestContainer_Resolve_With_Unsupported_Receiver_It_Should_Fail(t *testing.T) {
	err := instance.Resolve("STRING!")
	assert.EqualError(t, err, "container: invalid abstraction")
}

func TestContainer_Resolve_With_NonReference_Receiver_It_Should_Fail(t *testing.T) {
	var s Shape
	err := instance.Resolve(s)
	assert.EqualError(t, err, "container: invalid abstraction")
}

func TestContainer_Resolve_With_UnBounded_Reference_It_Should_Fail(t *testing.T) {
	instance.Reset()

	var s Shape
	err := instance.Resolve(&s)
	assert.EqualError(t, err, "container: no concrete found for: service.Shape")
}

func TestContainer_Fill_With_Struct_Pointer(t *testing.T) {
	err := instance.Singleton(func() Shape {
		return &Circle{a: 5}
	})
	assert.NoError(t, err)

	err = instance.NamedSingleton("C", func() Shape {
		return &Circle{a: 5}
	})
	assert.NoError(t, err)

	err = instance.Singleton(func() Database {
		return &MySQL{}
	})
	assert.NoError(t, err)

	myApp := struct {
		S Shape    `container:"inject"`
		D Database `container:"inject"`
		C Shape    `container:"inject"`
		X string
	}{}

	err = instance.Make(&myApp)
	assert.NoError(t, err)

	assert.IsType(t, &Circle{}, myApp.S)
	assert.IsType(t, &MySQL{}, myApp.D)
}

func TestContainer_Fill_Unexported_With_Struct_Pointer(t *testing.T) {
	err := instance.Singleton(func() Shape {
		return &Circle{a: 5}
	})
	assert.NoError(t, err)

	err = instance.Singleton(func() Database {
		return &MySQL{}
	})
	assert.NoError(t, err)

	myApp := struct {
		s Shape    `container:"inject"`
		d Database `container:"inject"`
		y int
	}{}

	err = instance.Make(&myApp)
	assert.NoError(t, err)

	assert.IsType(t, &Circle{}, myApp.s)
	assert.IsType(t, &MySQL{}, myApp.d)
}

func TestContainer_Fill_With_Invalid_Field_It_Should_Fail(t *testing.T) {
	err := instance.NamedSingleton("C", func() Shape {
		return &Circle{a: 5}
	})
	assert.NoError(t, err)

	type App struct {
		S string `container:"inject"`
	}

	myApp := App{}

	err = instance.Make(&myApp)
	assert.EqualError(t, err, "container: cannot make S field")
}

func TestContainer_Fill_With_Invalid_Tag_It_Should_Fail(t *testing.T) {
	type App struct {
		S string `container:"invalid"`
	}

	myApp := App{}

	err := instance.Make(&myApp)
	assert.EqualError(t, err, "container: S has an invalid struct tag")
}

func TestContainer_Fill_With_Invalid_Field_Name_It_Should_Fail(t *testing.T) {
	type App struct {
		S string `container:"inject"`
	}

	myApp := App{}

	err := instance.Make(&myApp)
	assert.EqualError(t, err, "container: cannot make S field")
}

func TestContainer_Fill_With_Invalid_Struct_It_Should_Fail(t *testing.T) {
	invalidStruct := 0
	err := instance.Make(&invalidStruct)
	assert.EqualError(t, err, "container: invalid structure")
}

func TestContainer_Fill_With_Invalid_Pointer_It_Should_Fail(t *testing.T) {
	var s Shape
	err := instance.Make(s)
	assert.EqualError(t, err, "container: invalid structure")
}

func TestContainer_Fill_With_Dependency_Missing_In_Chain(t *testing.T) {
	var instance = NewContainer()
	err := instance.Singleton(func() Shape {
		return &Circle{a: 5}
	})
	assert.NoError(t, err)

	err = instance.NamedSingletonLazy("C", func() (Shape, error) {
		var s Shape
		if err := instance.NamedResolve(&s, "foo"); err != nil {
			return nil, err
		}
		return &Circle{a: 5}, nil
	})
	assert.NoError(t, err)

	err = instance.Singleton(func() Database {
		return &MySQL{}
	})
	assert.NoError(t, err)

	myApp := struct {
		S Shape    `container:"inject"`
		D Database `container:"inject"`
		C Shape    `container:"inject"`
		X string
	}{}

	err = instance.Make(&myApp)
	assert.EqualError(t, err, "container: no concrete found for: service.Shape")
}
