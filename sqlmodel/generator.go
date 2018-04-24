package sqlmodel

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/go-openapi/inflect"
	"golang.org/x/tools/imports"
)

var _ ModelGenerator = &Generator{}

// GeneratorConfig controls how the code generation happens
type GeneratorConfig struct {
	// KeepSchema controlls whether the database schema to be kept as package
	KeepSchema bool
	// InlcudeDoc determines whether to include documentation
	InlcudeDoc bool
	// IgnoreTables ecludes the those tables from generation
	IgnoreTables []string
}

// Generator generates Golang structs from database schema
type Generator struct {
	// TagBuilder builds struct tags from column type
	TagBuilder TagBuilder
	// Config controls how the code generation happens
	Config *GeneratorConfig
}

// Generate generates the golang structs from database schema
func (g *Generator) Generate(pkg string, schema *Schema) (io.Reader, error) {
	buffer := &bytes.Buffer{}
	tables := g.tables(schema)

	if len(tables) == 0 {
		return buffer, nil
	}

	g.writePackage(pkg, schema.Name, buffer)

	for _, table := range tables {
		g.writeTable(pkg, schema.IsDefault, &table, buffer)
	}

	if err := g.format(buffer); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (g *Generator) tables(schema *Schema) []Table {
	tables := []Table{}
	ignore := g.ignore()

	for _, table := range schema.Tables {
		if index := sort.SearchStrings(ignore, table.Name); index >= 0 && index < len(ignore) {
			continue
		}

		tables = append(tables, table)
	}
	return tables
}

func (g *Generator) writePackage(pkg, name string, buffer io.Writer) {
	if g.Config.InlcudeDoc {
		fmt.Fprintf(buffer, "// Package %s contains an object model of database schema '%s'", pkg, name)
		fmt.Fprintln(buffer)
		fmt.Fprintln(buffer, "// Auto-generated at", time.Now().Format(time.UnixDate))
	}

	fmt.Fprintf(buffer, "package ")
	fmt.Fprintf(buffer, pkg)
	fmt.Fprintln(buffer)
}

func (g *Generator) writeTable(pkg string, isDefaultSchema bool, table *Table, buffer io.Writer) {
	columns := table.Columns
	length := len(columns)
	typeName := g.tableName(pkg, isDefaultSchema, table)

	if g.Config.InlcudeDoc {
		fmt.Fprintln(buffer)
		fmt.Fprintf(buffer, "// %s represents a data base table '%s'", typeName, table.Name)
		fmt.Fprintln(buffer)
	}

	fmt.Fprintf(buffer, "type %v struct {", typeName)
	fmt.Fprintln(buffer)

	for index, column := range columns {
		current := column
		fieldName := g.fieldName(&current)
		fieldType := g.fieldType(&current)
		fieldTag := g.TagBuilder.Build(&current)

		if g.Config.InlcudeDoc {
			if index > 0 {
				fmt.Fprintln(buffer)
			}
			fmt.Fprintf(buffer, "// %s represents a database column '%s' of type '%v'", fieldName, column.Name, column.Type)
			fmt.Fprintln(buffer)
		}

		fmt.Fprint(buffer, fieldName)
		fmt.Fprint(buffer, " ")
		fmt.Fprint(buffer, fieldType)
		fmt.Fprint(buffer, " ")
		fmt.Fprint(buffer, fieldTag)

		fmt.Fprintln(buffer)

		if index == length-1 {
			fmt.Fprintln(buffer, "}")
		}
	}
}

func (g *Generator) ignore() []string {
	ignore := g.Config.IgnoreTables

	if !sort.StringsAreSorted(ignore) {
		sort.Strings(ignore)
	}

	return ignore
}

func (g *Generator) tableName(pkg string, isDefaultSchema bool, table *Table) string {
	name := inflect.Camelize(table.Name)
	name = inflect.Singularize(name)

	if !g.Config.KeepSchema && !isDefaultSchema {
		pkg = inflect.Camelize(pkg)
		name = fmt.Sprintf("%s%s", pkg, name)
	}

	return name
}

func (g *Generator) fieldName(column *Column) string {
	name := inflect.Camelize(column.Name)
	name = strings.Replace(name, "Id", "ID", -1)
	return name
}

func (g *Generator) fieldType(column *Column) string {
	return column.ScanType
}

func (g *Generator) format(buffer *bytes.Buffer) error {
	data, err := imports.Process("model", buffer.Bytes(), nil)
	if err != nil {
		return err
	}

	data, err = format.Source(data)
	if err != nil {
		return err
	}

	buffer.Reset()

	_, err = buffer.Write(data)
	return err
}
