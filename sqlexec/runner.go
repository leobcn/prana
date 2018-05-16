package sqlexec

import "database/sql"

// Runner runs a SQL statement for given command name and parameters.
type Runner struct {
	// DriverName is the current SQL driver
	DriverName string
	// FileSystem represents the project directory file system.
	FileSystem FileSystem
	// DB is a client to underlying database.
	DB *sql.DB
}

// Run runs a given command with provided parameters.
func (r *Runner) Run(name string, args ...Param) (*sql.Rows, error) {
	provider := &Provider{
		DriverName: r.DriverName,
	}

	if err := provider.ReadDir(r.FileSystem); err != nil {
		return nil, err
	}

	query, err := provider.Query(name)
	if err != nil {
		return nil, err
	}

	stmt, err := r.DB.Prepare(query)
	if err != nil {
		return nil, err
	}

	defer func() {
		if stmtErr := stmt.Close(); err == nil {
			err = stmtErr
		}
	}()

	return stmt.Query(args...)
}
