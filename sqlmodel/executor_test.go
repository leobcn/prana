package sqlmodel_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/phogolabs/prana/fake"
	"github.com/phogolabs/prana/sqlmodel"
)

var _ = Describe("Executor", func() {
	var (
		executor  *sqlmodel.Executor
		spec      *sqlmodel.Spec
		provider  *fake.ModelProvider
		composer  *fake.ModelComposer
		reader    *fake.Buffer
		schemaDef *sqlmodel.Schema
	)

	BeforeEach(func() {
		schemaDef = &sqlmodel.Schema{
			Name:      "public",
			IsDefault: true,
			Tables: []sqlmodel.Table{
				{
					Name: "table1",
					Columns: []sqlmodel.Column{
						{
							Name:     "ID",
							ScanType: "string",
						},
					},
				},
			},
		}

		dir, err := ioutil.TempDir("", "prana")
		Expect(err).To(BeNil())

		spec = &sqlmodel.Spec{
			Schema: "public",
			Tables: []string{"table1"},
			Dir:    filepath.Join(dir, "entity"),
		}

		reader = &fake.Buffer{}

		provider = &fake.ModelProvider{}
		provider.TablesReturns([]string{"table1"}, nil)
		provider.SchemaReturns(schemaDef, nil)

		composer = &fake.ModelComposer{}
		composer.ComposeReturns(reader, nil)

		executor = &sqlmodel.Executor{
			Config: &sqlmodel.ExecutorConfig{
				KeepSchema: true,
			},
			Provider: provider,
			Composer: composer,
		}
	})

	Describe("Write", func() {
		BeforeEach(func() {
			reader := bytes.NewBufferString("source")
			composer.ComposeReturns(reader, nil)
		})

		It("writes the generated source successfully", func() {
			writer := &bytes.Buffer{}
			Expect(executor.Write(writer, spec)).To(Succeed())
			Expect(writer.String()).To(Equal("source"))

			Expect(provider.TablesCallCount()).To(BeZero())
			Expect(provider.SchemaCallCount()).To(Equal(1))

			schemaName, tables := provider.SchemaArgsForCall(0)
			Expect(schemaName).To(Equal("public"))
			Expect(tables).To(ContainElement("table1"))

			Expect(composer.ComposeCallCount()).To(Equal(1))
			packageName, schemaDefintion := composer.ComposeArgsForCall(0)

			Expect(packageName).To(Equal("entity"))
			Expect(schemaDefintion).To(Equal(schemaDef))
		})

		Context("when the schema is not default", func() {
			BeforeEach(func() {
				schemaDef.IsDefault = false
			})

			It("uses the schema name as package name", func() {
				Expect(executor.Write(ioutil.Discard, spec)).To(Succeed())
				Expect(composer.ComposeCallCount()).To(Equal(1))
				packageName, _ := composer.ComposeArgsForCall(0)

				Expect(packageName).To(Equal(schemaDef.Name))
			})
		})

		Context("when the tables are not provided", func() {
			BeforeEach(func() {
				spec.Tables = []string{}
			})

			It("writes the generated source successfully", func() {
				writer := &bytes.Buffer{}
				reader := bytes.NewBufferString("source")
				composer.ComposeReturns(reader, nil)

				Expect(executor.Write(writer, spec)).To(Succeed())
				Expect(writer.String()).To(Equal("source"))

				Expect(provider.TablesCallCount()).To(Equal(1))
				Expect(provider.TablesArgsForCall(0)).To(Equal("public"))

				Expect(provider.SchemaCallCount()).To(Equal(1))

				schemaName, tables := provider.SchemaArgsForCall(0)
				Expect(schemaName).To(Equal("public"))
				Expect(tables).To(ContainElement("table1"))

				Expect(composer.ComposeCallCount()).To(Equal(1))
				packageName, schemaDefintion := composer.ComposeArgsForCall(0)

				Expect(packageName).To(Equal("entity"))
				Expect(schemaDefintion).To(Equal(schemaDef))
			})

			Context("when getting the schema tables fails", func() {
				BeforeEach(func() {
					provider.TablesReturns([]string{}, fmt.Errorf("Oh no!"))
				})

				It("returns the error", func() {
					writer := &bytes.Buffer{}
					Expect(executor.Write(writer, spec)).To(MatchError("Oh no!"))
					Expect(writer.Bytes()).To(BeEmpty())
				})
			})
		})

		Context("when the composer fails", func() {
			BeforeEach(func() {
				composer.ComposeReturns(nil, fmt.Errorf("Oh no!"))
			})

			It("returns the error", func() {
				Expect(executor.Write(ioutil.Discard, spec)).To(MatchError("Oh no!"))
			})
		})

		Context("when the copy fails", func() {
			BeforeEach(func() {
				reader.ReadReturns(0, fmt.Errorf("Oh no!"))
				composer.ComposeReturns(reader, nil)
			})

			It("returns the error", func() {
				Expect(executor.Write(ioutil.Discard, spec)).To(MatchError("Oh no!"))
			})
		})
	})

	Describe("Create", func() {
		BeforeEach(func() {
			composer.ComposeReturns(bytes.NewBufferString("source"), nil)
		})

		ItCreatesTheSchemaInRootPkg := func(filename, pkg string) {
			It("creates a package with generated source successfully", func() {
				path, err := executor.Create(spec)
				Expect(err).To(Succeed())

				Expect(spec.Dir).To(BeADirectory())
				Expect(filepath.Join(spec.Dir, filename)).To(BeARegularFile())
				Expect(path).To(Equal(filepath.Join(spec.Dir, filename)))

				Expect(provider.TablesCallCount()).To(BeZero())
				Expect(provider.SchemaCallCount()).To(Equal(1))

				schemaName, tables := provider.SchemaArgsForCall(0)
				Expect(schemaName).To(Equal("public"))
				Expect(tables).To(ContainElement("table1"))

				Expect(composer.ComposeCallCount()).To(Equal(1))
				packageName, schemaDefintion := composer.ComposeArgsForCall(0)

				Expect(packageName).To(Equal(pkg))
				Expect(schemaDefintion).To(Equal(schemaDef))
			})
		}

		ItCreatesTheSchemaInRootPkg("schema.go", "entity")

		Context("when the schema is not default", func() {
			BeforeEach(func() {
				schemaDef.IsDefault = false
			})

			ItCreatesTheSchemaInRootPkg("public/schema.go", "public")
		})

		Context("when the KeepSchema is false", func() {
			BeforeEach(func() {
				executor.Config.KeepSchema = false
			})

			ItCreatesTheSchemaInRootPkg("schema.go", "entity")

			Context("when the schema is not default", func() {
				BeforeEach(func() {
					schemaDef.IsDefault = false
				})

				ItCreatesTheSchemaInRootPkg("public.go", "entity")
			})
		})

		Context("when the tables are not provided", func() {
			BeforeEach(func() {
				spec.Tables = []string{}
			})

			It("creates a package with generated source successfully", func() {
				path, err := executor.Create(spec)
				Expect(err).To(Succeed())

				Expect(spec.Dir).To(BeADirectory())
				Expect(filepath.Join(spec.Dir, "schema.go")).To(BeARegularFile())
				Expect(path).To(Equal(filepath.Join(spec.Dir, "schema.go")))

				Expect(provider.TablesCallCount()).To(Equal(1))
				Expect(provider.TablesArgsForCall(0)).To(Equal("public"))
				Expect(provider.SchemaCallCount()).To(Equal(1))

				schemaName, tables := provider.SchemaArgsForCall(0)
				Expect(schemaName).To(Equal("public"))
				Expect(tables).To(ContainElement("table1"))

				Expect(composer.ComposeCallCount()).To(Equal(1))
				packageName, schemaDefintion := composer.ComposeArgsForCall(0)

				Expect(packageName).To(Equal("entity"))
				Expect(schemaDefintion).To(Equal(schemaDef))
			})

			Context("when the provider fails to get table names", func() {
				BeforeEach(func() {
					provider.TablesReturns([]string{}, fmt.Errorf("Oh no!"))
				})

				It("returns the error", func() {
					path, err := executor.Create(spec)
					Expect(err).To(MatchError("Oh no!"))
					Expect(path).To(BeEmpty())
				})
			})
		})

		Context("when the spec schema is not default", func() {
			BeforeEach(func() {
				schemaDef.IsDefault = false
			})

			It("creates a package with generated source successfully", func() {
				path, err := executor.Create(spec)
				Expect(err).To(Succeed())
				Expect(spec.Dir).To(BeADirectory())

				Expect(filepath.Join(spec.Dir, "public", "schema.go")).To(BeARegularFile())
				Expect(path).To(Equal(filepath.Join(spec.Dir, "public", "schema.go")))
			})
		})

		Context("when getting the schame fails", func() {
			BeforeEach(func() {
				provider.SchemaReturns(nil, fmt.Errorf("Oh no!"))
			})

			It("returns the error", func() {
				path, err := executor.Create(spec)
				Expect(err).To(MatchError("Oh no!"))
				Expect(path).To(BeEmpty())
			})
		})

		Context("when the reader has empty content", func() {
			BeforeEach(func() {
				composer.ComposeReturns(&bytes.Buffer{}, nil)
			})

			It("creates a package with generated source successfully", func() {
				path, err := executor.Create(spec)
				Expect(err).To(Succeed())
				Expect(path).To(BeEmpty())
			})
		})

		Context("when the composer fails", func() {
			BeforeEach(func() {
				composer.ComposeReturns(nil, fmt.Errorf("Oh no!"))
			})

			It("returns the error", func() {
				path, err := executor.Create(spec)
				Expect(err).To(MatchError("Oh no!"))
				Expect(path).To(BeEmpty())
			})
		})

		Context("when creating the dir fails", func() {
			BeforeEach(func() {
				spec.Dir = "/mydir"
			})

			It("returns the error", func() {
				path, err := executor.Create(spec)
				Expect(err).To(MatchError("mkdir /mydir: permission denied"))
				Expect(path).To(BeEmpty())
			})
		})

		Context("when the copy fails", func() {
			BeforeEach(func() {
				reader.ReadReturns(0, fmt.Errorf("Oh no!"))
				composer.ComposeReturns(reader, nil)
			})

			It("returns the error", func() {
				path, err := executor.Create(spec)
				Expect(err).To(MatchError("Oh no!"))
				Expect(path).To(BeEmpty())
			})
		})
	})
})
