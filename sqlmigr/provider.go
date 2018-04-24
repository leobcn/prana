package sqlmigr

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
)

var _ MigrationProvider = &Provider{}

// Provider provides all migration for given project.
type Provider struct {
	// FileSystem represents the project directory file system.
	FileSystem FileSystem
	// DB is a client to underlying database.
	DB *sqlx.DB
}

// Migrations returns the project migrations.
func (m *Provider) Migrations() ([]Migration, error) {
	local := []Migration{}

	err := m.FileSystem.Walk("/", func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return fmt.Errorf("Directory '%s' does not exist", path)
		}

		if info.IsDir() {
			return nil
		}

		matched, err := filepath.Match("*.sql", info.Name())
		if err != nil || !matched {
			return err
		}

		sqlmigr, err := Parse(path)
		if err != nil {
			return err
		}

		local = append(local, *sqlmigr)
		return nil
	})

	if err != nil {
		return []Migration{}, err
	}

	remote := []Migration{}

	query := &bytes.Buffer{}
	query.WriteString("SELECT id, description, created_at ")
	query.WriteString("FROM migrations ")
	query.WriteString("ORDER BY id ASC")

	if err := m.DB.Select(&remote, query.String()); err != nil {
		return []Migration{}, err
	}

	return m.merge(remote, local)
}

// Insert inserts executed sqlmigr item in the sqlmigrs table.
func (m *Provider) Insert(item *Migration) error {
	item.CreatedAt = time.Now()

	query := &bytes.Buffer{}
	query.WriteString("INSERT INTO migrations(id, description, created_at) ")
	query.WriteString("VALUES (?, ?, ?)")

	if _, err := m.DB.Exec(query.String(), item.ID, item.Description, item.CreatedAt); err != nil {
		return err
	}

	return nil
}

// Delete deletes applied sqlmigr item from sqlmigrs table.
func (m *Provider) Delete(item *Migration) error {
	query := &bytes.Buffer{}
	query.WriteString("DELETE FROM migrations ")
	query.WriteString("WHERE id = ?")

	if _, err := m.DB.Exec(query.String(), item.ID); err != nil {
		return err
	}

	return nil
}

// Exists returns true if the sqlmigr exists
func (m *Provider) Exists(item *Migration) bool {
	count := 0

	if err := m.DB.Get(&count, "SELECT count(id) FROM migrations WHERE id = ?", item.ID); err != nil {
		return false
	}

	return count == 1
}

func (m *Provider) merge(remote, local []Migration) ([]Migration, error) {
	result := local

	for index, r := range remote {
		l := local[index]

		if r.ID != l.ID {
			return []Migration{}, fmt.Errorf("Mismatched migration id. Expected: '%s' but has '%s'", r.ID, l.ID)
		}

		if r.Description != l.Description {
			return []Migration{}, fmt.Errorf("Mismatched migration description. Expected: '%s' but has '%s'", r.Description, l.Description)
		}

		result[index] = r
	}

	return result, nil
}