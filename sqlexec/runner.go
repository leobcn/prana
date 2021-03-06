package sqlexec

import "github.com/jmoiron/sqlx"

// Runner runs a SQL statement for given command name and parameters.
type Runner struct {
	// FileSystem represents the project directory file system.
	FileSystem FileSystem
	// DB is a client to underlying database.
	DB *sqlx.DB
}

// Run runs a given command with provided parameters.
func (r *Runner) Run(name string, args ...Param) (*Rows, error) {
	provider := &Provider{}

	if err := provider.ReadDir(r.FileSystem); err != nil {
		return nil, err
	}

	query, err := provider.Query(name)
	if err != nil {
		return nil, err
	}

	stmt, err := r.DB.Preparex(query)
	if err != nil {
		return nil, err
	}

	defer func() {
		if stmtErr := stmt.Close(); err == nil {
			err = stmtErr
		}
	}()

	return stmt.Queryx(args...)
}
