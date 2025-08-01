// Copyright 2025 Pinterest
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package checks_test

import (
	"testing"

	"github.com/pinterest/thriftcheck/checks"
	"go.uber.org/thriftrw/ast"
)

func TestCheckSetValueType(t *testing.T) {
	tests := []Test{
		{
			node: ast.SetType{ValueType: ast.BaseType{ID: ast.StringTypeID}},
			want: []string{},
		},
		{
			node: ast.SetType{ValueType: ast.SetType{ValueType: ast.BaseType{ID: ast.StringTypeID}}},
			want: []string{
				`t.thrift:0:1: error: set value must be a primitive type (set.value.type)`,
			},
		},
		{
			prog: &ast.Program{},
			node: ast.SetType{ValueType: ast.TypeReference{Name: "Enum"}},
			want: []string{
				`t.thrift:0:1: error: set value must be a primitive type (set.value.type)`,
			},
		},
		{
			prog: &ast.Program{Definitions: []ast.Definition{
				&ast.Enum{Name: "Enum"},
			}},
			node: ast.SetType{ValueType: ast.TypeReference{Name: "Enum"}},
			want: []string{},
		},
	}

	check := checks.CheckSetValueType()
	RunTests(t, &check, tests)
}
