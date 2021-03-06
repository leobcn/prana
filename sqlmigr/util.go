package sqlmigr

import "github.com/jmoiron/sqlx"

// RunAll runs all sqlmigrs
func RunAll(db *sqlx.DB, fileSystem FileSystem) error {
	executor := &Executor{
		Provider: &Provider{
			FileSystem: fileSystem,
			DB:         db,
		},
		Runner: &Runner{
			FileSystem: fileSystem,
			DB:         db,
		},
		Generator: &Generator{
			FileSystem: fileSystem,
		},
	}

	_, err := executor.RunAll()
	return err
}
