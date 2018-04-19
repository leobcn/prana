package schema_test

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/phogolabs/oak/fake"
	"github.com/phogolabs/oak/schema"
	"golang.org/x/tools/imports"
)

var _ = Describe("Generator", func() {
	var (
		generator *schema.Generator
		builder   *fake.SchemaTagBuilder
		schemaDef *schema.Schema
	)

	BeforeEach(func() {
		schemaDef = &schema.Schema{
			Name: "schema",
			Tables: []schema.Table{
				{
					Name: "table1",
					Columns: []schema.Column{
						{
							Name:     "id",
							ScanType: "string",
							Type: schema.ColumnType{
								Name:          "varchar",
								IsPrimaryKey:  true,
								IsNullable:    true,
								CharMaxLength: 200,
							},
						},
						{
							Name:     "name",
							ScanType: "string",
							Type: schema.ColumnType{
								Name:          "varchar",
								IsPrimaryKey:  false,
								IsNullable:    false,
								CharMaxLength: 200,
							},
						},
					},
				},
			},
		}

		builder = &fake.SchemaTagBuilder{}
		builder.BuildReturns("`db`")
		generator = &schema.Generator{
			TagBuilder: builder,
			Config: &schema.GeneratorConfig{
				InlcudeDoc: false,
			},
		}
	})

	It("generates the schema successfully", func() {
		source := &bytes.Buffer{}
		fmt.Fprintln(source, "package model")
		fmt.Fprintln(source)
		fmt.Fprintln(source, "type Table1 struct {")
		fmt.Fprintln(source, "        Id string `db`")
		fmt.Fprintln(source, "        Name string `db`")
		fmt.Fprintln(source, "}")

		data, err := imports.Process("model", source.Bytes(), nil)
		Expect(err).To(BeNil())

		data, err = format.Source(data)
		Expect(err).To(BeNil())

		reader, err := generator.Compose("model", schemaDef)
		Expect(err).To(BeNil())

		Expect(builder.BuildCallCount()).To(Equal(2))
		Expect(builder.BuildArgsForCall(0)).To(Equal(&schemaDef.Tables[0].Columns[0]))
		Expect(builder.BuildArgsForCall(1)).To(Equal(&schemaDef.Tables[0].Columns[1]))

		generated, err := ioutil.ReadAll(reader)
		Expect(err).To(BeNil())
		Expect(generated).To(Equal(data))
	})

	Context("when the table is ignored", func() {
		BeforeEach(func() {
			generator.Config.IgnoreTables = []string{"table1"}
		})

		It("generates the schema successfully", func() {
			reader, err := generator.Compose("model", schemaDef)
			Expect(err).To(BeNil())

			data, err := ioutil.ReadAll(reader)
			Expect(err).To(BeNil())
			Expect(data).To(BeEmpty())
		})
	})

	Context("when including documentation is disabled", func() {
		BeforeEach(func() {
			generator.Config.InlcudeDoc = true
		})

		It("generates the schema successfully", func() {
			reader, err := generator.Compose("model", schemaDef)
			Expect(err).To(BeNil())

			data, err := ioutil.ReadAll(reader)
			Expect(err).To(BeNil())

			source := string(data)
			Expect(source).To(ContainSubstring("// Table1 represents a data base table 'table1'"))
			Expect(source).To(ContainSubstring("// Id represents a database column 'id' of type 'VARCHAR(200) PRIMARY KEY NULL'"))
		})
	})

	Context("when the tables are ignored", func() {
		BeforeEach(func() {
			generator.Config.IgnoreTables = []string{"table1", "atab"}
		})

		It("generates the schema successfully", func() {
			reader, err := generator.Compose("model", schemaDef)
			Expect(err).To(BeNil())

			data, err := ioutil.ReadAll(reader)
			Expect(err).To(BeNil())
			Expect(data).To(BeEmpty())
		})
	})

	Context("when no tables are provided", func() {
		BeforeEach(func() {
			schemaDef.Tables = []schema.Table{}
		})

		It("generates the schema successfully", func() {
			reader, err := generator.Compose("model", schemaDef)
			Expect(err).To(BeNil())

			data, err := ioutil.ReadAll(reader)
			Expect(err).To(BeNil())
			Expect(data).To(BeEmpty())
		})
	})

	Context("when the package name is not provided", func() {
		It("returns an error", func() {
			reader, err := generator.Compose("", schemaDef)
			Expect(reader).To(BeNil())
			Expect(err).To(MatchError("model:2:1: expected 'IDENT', found 'type'"))
		})
	})
})
