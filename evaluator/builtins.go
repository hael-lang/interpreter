package evaluator

import (
	"fmt"
	"hael/object"
	"net/http"
	"os"
	"path/filepath"
)

var Builtins map[string]*object.Builtin

func InitBuiltins(env *object.Environment) {
	Builtins = map[string]*object.Builtin{
		"len": &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return newError("wrong number of arguments. got=%d, want=1", len(args))
				}

				switch arg := args[0].(type) {
				case *object.Array:
					return &object.Integer{Value: int64(len(arg.Elements))}
				case *object.String:
					return &object.Integer{Value: int64(len(arg.Value))}
				default:
					return newError("argument to `len` not supported, got %s", args[0].Type())
				}
			},
		},
		"push": &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return newError("wrong number of arguments. got=%d, want=2", len(args))
				}

				if args[0].Type() != object.ARRAY_OBJ {
					return newError("argument to `push` must be ARRAY, got %s", args[0].Type())
				}

				arr := args[0].(*object.Array)
				length := len(arr.Elements)

				newElements := make([]object.Object, length+1)
				copy(newElements, arr.Elements)
				newElements[length] = args[1]

				return &object.Array{Elements: newElements}
			},
		},
		"pop": &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return newError("wrong number of arguments. got=%d, want=1", len(args))
				}

				if args[0].Type() != object.ARRAY_OBJ {
					return newError("argument to `pop` must be ARRAY, got %s", args[0].Type())
				}

				arr := args[0].(*object.Array)
				if len(arr.Elements) == 0 {
					return newError("cannot pop from empty array")
				}

				lastElement := arr.Elements[len(arr.Elements)-1]

				newElements := make([]object.Object, len(arr.Elements)-1)
				copy(newElements, arr.Elements[:len(arr.Elements)-1])
				arr.Elements = newElements

				return lastElement
			},
		},
		"type": &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return newError("type expects one argument")
				}
				return &object.String{Value: string(args[0].Type())}
			},
		},
		"slice": &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return newError("wrong number of arguments. got=%d, want=2 or 3", len(args))
				}

				if args[0].Type() != object.ARRAY_OBJ {
					return newError("first argument to `slice` must be ARRAY, got %s", args[0].Type())
				}

				if args[1].Type() != object.INTEGER_OBJ {
					return newError("second argument to `slice` must be INTEGER, got %s", args[1].Type())
				}

				end := int64(len(args[0].(*object.Array).Elements))

				if len(args) == 3 {
					if args[2].Type() != object.INTEGER_OBJ {
						return newError("third argument to `slice` must be INTEGER, got %s", args[2].Type())
					}
					end = args[2].(*object.Integer).Value
				}

				arr := args[0].(*object.Array)
				start := args[1].(*object.Integer).Value

				if start < 0 || end > int64(len(arr.Elements)) || start > end {
					return newError("invalid slice indices: start=%d, end=%d", start, end)
				}

				newElements := make([]object.Object, end-start)
				copy(newElements, arr.Elements[start:end])

				return &object.Array{Elements: newElements}
			},
		},
		"print": &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				for _, text := range args {
					println(text.Inspect())
				}

				return NULL
			},
		},
		"do": &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				fn, ok := args[0].(*object.Function)
				if ok {
					return Eval(fn.Body, env)
				}

				return NULL
			},
		},
		"listenHTTP": &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return newError("wrong number of arguments. got=%d, want=2", len(args))
				}

				port, ok := args[0].(*object.String)
				if !ok {
					return newError("1st argument to `listenHTTP` must be STRING, got %s", args[0].Type())
				}

				hash, ok := args[1].(*object.Hash)
				if !ok {
					return newError("2nd argument to `listenHTTP` must be HASH, got %s", args[1].Type())
				}

				mux := http.NewServeMux()

				for _, pair := range hash.Pairs {
					path, ok := pair.Key.(*object.String)

					if !ok {
						return newError("route key must be STRING, got %s", pair.Key.Type())
					}

					fn, ok := pair.Value.(*object.Function)
					if !ok {
						return newError("route handler for %s must be FUNCTION, got %s", path.Value, pair.Value.Type())
					}

					mux.HandleFunc(path.Value, func(w http.ResponseWriter, r *http.Request) {
						contentType := "text/html"
						switch filepath.Ext(r.URL.Path) {
						case ".wasm":
							contentType = "application/wasm"
						case ".js":
							contentType = "application/javascript"
						}
						w.Header().Set("Content-Type", contentType)

						result := Eval(fn.Body, env)

						if result.Type() == object.ERROR_OBJ {
							http.Error(w, result.Inspect(), http.StatusInternalServerError)
							return
						}

						fmt.Fprintf(w, "%s", result.Inspect())
					})
				}

				http.ListenAndServe(":"+port.Value, mux)

				return GOOD
			},
		},
		"parseHTML": &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return newError("wrong number of arguments. got=%d, want=1", len(args))
				}

				path, ok := args[0].(*object.String)
				if !ok {
					return newError("argument to `parseHTML` must be STRING, got %s", args[0].Type())
				}

				content, err := os.ReadFile(path.Value)
				if err != nil {
					return newError("failed to read file %s: %v", path.Value, err)
				}

				return &object.String{Value: string(content)}
			},
		},
	}
}
