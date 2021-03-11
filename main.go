package main

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"

	"go.uber.org/zap"
)

var log, _ = zap.NewDevelopment()

func main() {
	defer log.Sync()
	var analyzer = &analysis.Analyzer{
		Name: "rvalue",
		Doc:  "reports nil return",
		Run:  rValue,
	}
	singlechecker.Main(analyzer)
}

type visitor func(ast.Node) bool

func (f visitor) Visit(node ast.Node) ast.Visitor {
	if f(node) {
		return f
	}
	return nil
}

func rValue(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			var fDecl, ok = n.(*ast.FuncDecl)
			if !ok {
				return true
			}

			var f = fDecl.Type
			if f.Results == nil {
				return false
			}

			var maybeNil []int
			var errPos = -1
			for i, r := range f.Results.List {
				var t = pass.TypesInfo.TypeOf(r.Type)
				if t == nil {
					log.Error("Can't get type of return value")
					continue
				}
				switch v := t.Underlying().(type) {
				case *types.Interface:
					if v.NumMethods() == 1 && v.Method(0).FullName() == "(error).Error" {
						errPos = i
					} else {
						maybeNil = append(maybeNil, i)
					}
				case *types.Pointer:
					maybeNil = append(maybeNil, i)
				}
			}
			if len(maybeNil) == 0 {
				return false
			}

			if errPos == -1 {
				var fDecl = *fDecl
				fDecl.Body = nil
				pass.Reportf(n.Pos(), "Function `%s` should return `error`", stringifyNode(pass.Fset, &fDecl))
				return false
			}

			ast.Walk(visitor(func(n ast.Node) bool {
				var r, ok = n.(*ast.ReturnStmt)
				if !ok {
					return true
				}
				if isExprNil(r.Results[errPos]) {
					for _, i := range maybeNil {
						if isExprNil(r.Results[i]) {
							pass.Reportf(n.Pos(), "Return value of `%s` at %d in `%s` should not be `nil`", fDecl.Name.Name, i, stringifyNode(pass.Fset, r))
						}
					}
					return false
				}
				return true
			}), fDecl.Body)

			return false
		})
	}

	return nil, nil
}

func isExprNil(expr ast.Expr) bool {
	var t, ok = expr.(*ast.Ident)
	return ok && t.Name == "nil"
}

func stringifyNode(fset *token.FileSet, node interface{}) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		log.Fatal("Failed to render", zap.Error(err))
	}
	return buf.String()
}
