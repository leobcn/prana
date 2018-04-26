package sqlexec_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/phogolabs/prana/sqlexec"
)

var _ = Describe("Command", func() {
	It("prepares the command correctly", func() {
		stmt := sqlexec.SQL("SELECT * FROM users WHERE id = ?", 1)
		query, params := stmt.Prepare()
		Expect(query).To(Equal("SELECT * FROM users WHERE id = :arg0"))
		Expect(params).To(HaveKeyWithValue("arg0", 1))
	})

	Context("when parameters are more than the required number", func() {
		It("returns the query", func() {
			stmt := sqlexec.SQL("SELECT * FROM users WHERE id = ? and name = ?", 2)
			query, params := stmt.Prepare()
			Expect(query).To(Equal("SELECT * FROM users WHERE id = :arg0 and name = :arg1"))
			Expect(params).To(HaveLen(1))
			Expect(params).To(HaveKeyWithValue("arg0", 2))
		})
	})
})
