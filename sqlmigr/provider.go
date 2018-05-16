package sqlmigr

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var _ MigrationProvider = &Provider{}

// Provider provides all migration for given project.
type Provider struct {
	// FileSystem represents the project directory file system.
	FileSystem FileSystem
	// DriverName returns the current driver name
	DriverName string
	// DB is a client to underlying database.
	DB *sql.DB
}

// Migrations returns the project migrations.
func (m *Provider) Migrations() ([]*Migration, error) {
	local, err := m.files()
	if err != nil {
		return local, err
	}

	remote, err := m.query()
	if err != nil {
		return remote, err
	}

	return m.merge(remote, local)
}

func (m *Provider) files() ([]*Migration, error) {
	local := []*Migration{}

	err := m.FileSystem.Walk("/", func(path string, info os.FileInfo, err error) error {
		if ferr := m.filter(info); ferr != nil {
			if ferr.Error() == "skip" {
				ferr = nil
			}

			return ferr
		}

		migration, err := Parse(path)
		if err != nil {
			return err
		}

		if !m.supported(migration.Drivers) {
			return nil
		}

		if index := len(local) - 1; index >= 0 {
			if prev := local[index]; migration.Equal(prev) {
				prev.Drivers = append(prev.Drivers, migration.Drivers...)
				local[index] = prev
				return nil
			}
		}

		local = append(local, migration)
		return nil
	})

	if err != nil {
		return []*Migration{}, err
	}

	return local, nil
}

func (m *Provider) filter(info os.FileInfo) error {
	skip := fmt.Errorf("skip")

	if info == nil {
		return os.ErrNotExist
	}

	if info.IsDir() {
		return skip
	}

	matched, _ := filepath.Match("*.sql", info.Name())

	if !matched {
		return skip
	}

	return nil
}

func (m *Provider) supported(drivers []string) bool {
	for _, driver := range drivers {
		if driver == every || driver == m.DriverName {
			return true
		}
	}

	return false
}

func (m *Provider) query() ([]*Migration, error) {
	query := &bytes.Buffer{}
	query.WriteString("SELECT id, description, created_at ")
	query.WriteString("FROM migrations ")
	query.WriteString("ORDER BY id ASC")

	rows, err := m.DB.Query(query.String())
	if err != nil {
		if IsNotExist(err) {
			err = nil
		}
		return []*Migration{}, err
	}

	defer rows.Close()

	remote := []*Migration{}

	for rows.Next() {
		migration := &Migration{}
		_ = migration.Scan(rows)
		remote = append(remote, migration)
	}

	return remote, nil
}

// Insert inserts executed sqlmigr item in the sqlmigrs table.
func (m *Provider) Insert(item *Migration) error {
	item.CreatedAt = time.Now()

	builder := &bytes.Buffer{}
	builder.WriteString("INSERT INTO migrations(id, description, created_at) ")
	builder.WriteString("VALUES (?, ?, ?)")

	if _, err := m.DB.Exec(builder.String(), item.ID, item.Description, item.CreatedAt); err != nil {
		return err
	}

	return nil
}

// Delete deletes applied sqlmigr item from sqlmigrs table.
func (m *Provider) Delete(item *Migration) error {
	builder := &bytes.Buffer{}
	builder.WriteString("DELETE FROM migrations ")
	builder.WriteString("WHERE id = ?")

	if _, err := m.DB.Exec(builder.String(), item.ID); err != nil {
		return err
	}

	return nil
}

// Exists returns true if the sqlmigr exists
func (m *Provider) Exists(item *Migration) bool {
	count := 0

	if err := m.DB.QueryRow("SELECT count(id) FROM migrations WHERE id = ?", item.ID).Scan(&count); err != nil {
		return false
	}

	return count == 1
}

func (m *Provider) merge(remote, local []*Migration) ([]*Migration, error) {
	result := local

	for index, r := range remote {
		l := local[index]

		if r.ID != l.ID {
			return []*Migration{}, fmt.Errorf("mismatched migration id. Expected: '%s' but has '%s'", r.ID, l.ID)
		}

		if r.Description != l.Description {
			return []*Migration{}, fmt.Errorf("mismatched migration description. Expected: '%s' but has '%s'", r.Description, l.Description)
		}

		// Merge creation time
		l.CreatedAt = r.CreatedAt
		result[index] = l
	}

	return result, nil
}
