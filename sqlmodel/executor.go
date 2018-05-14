package sqlmodel

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Executor executes the schema generation
type Executor struct {
	// ModelGenerator is the SQL model generator
	ModelGenerator Generator
	// QueryGenerator is the SQL script generator
	QueryGenerator Generator
	// Provider provides information the database schema
	Provider SchemaProvider
}

// Write writes the generated schema sqlmodels to a writer
func (e *Executor) Write(w io.Writer, spec *Spec) error {
	_, err := e.writeSchema(w, spec)
	return err
}

// Create creates a package with the generated schema sqlmodels
func (e *Executor) Create(spec *Spec) (string, error) {
	reader := &bytes.Buffer{}
	schema, err := e.writeSchema(reader, spec)
	if err != nil {
		return "", err
	}

	body, _ := ioutil.ReadAll(reader)
	if len(body) == 0 {
		return "", nil
	}

	filepath := e.fileOf(e.nameOf(schema), "schema.go")
	file, err := spec.FileSystem.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return "", err
	}

	defer func() {
		if ioErr := file.Close(); err == nil {
			err = ioErr
		}
	}()

	if _, err = file.Write(body); err != nil {
		return "", err
	}

	return filepath, nil
}

// CreateScript creates a model SQL routines
func (e *Executor) CreateScript(spec *Spec) (string, error) {
	schema, err := e.schemaOf(spec)
	if err != nil {
		return "", err
	}

	reader := &bytes.Buffer{}
	ctx := &GeneratorContext{
		Writer:  reader,
		Package: spec.Name,
		Schema:  schema,
	}

	if err = e.QueryGenerator.Generate(ctx); err != nil {
		return "", err
	}

	body, _ := ioutil.ReadAll(reader)
	if len(body) == 0 {
		return "", nil
	}

	filepath := e.fileOf(e.nameOf(schema), "routine.sql")
	file, err := spec.FileSystem.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return "", err
	}

	defer func() {
		if ioErr := file.Close(); err == nil {
			err = ioErr
		}
	}()

	if _, err = file.Write(body); err != nil {
		return "", err
	}

	return filepath, nil
}

func (e *Executor) writeSchema(w io.Writer, spec *Spec) (*Schema, error) {
	schema, err := e.schemaOf(spec)
	if err != nil {
		return nil, err
	}

	ctx := &GeneratorContext{
		Writer:  w,
		Package: spec.Name,
		Schema:  schema,
	}

	if err = e.ModelGenerator.Generate(ctx); err != nil {
		return nil, err
	}

	return schema, nil
}

func (e *Executor) schemaOf(spec *Spec) (*Schema, error) {
	if len(spec.Tables) == 0 {
		tables, err := e.Provider.Tables(spec.Schema)
		if err != nil {
			return nil, err
		}

		spec.Tables = tables
	}

	schema, err := e.Provider.Schema(spec.Schema, spec.Tables...)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

func (e *Executor) fileOf(schema, filename string) string {
	if schema != "" {
		filename = fmt.Sprintf("%s%s", schema, filepath.Ext(filename))
	}

	return filename
}

func (e *Executor) nameOf(schema *Schema) string {
	if !schema.IsDefault {
		return schema.Name
	}
	return ""
}
