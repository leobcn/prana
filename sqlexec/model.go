// Package sqlexec provides primitives and functions to work with raw SQL
// statements and pre-defined SQL Scripts.
package sqlexec

import (
	"github.com/jmoiron/sqlx"
	"github.com/phogolabs/parcello"
)

var (
	format = "20060102150405"
)

// Param is a command parameter for given query.
type Param = interface{}

// Rows is a wrapper around sql.Rows which caches costly reflect operations
// during a looped StructScan.
type Rows = sqlx.Rows

// FileSystem provides with primitives to work with the underlying file system
type FileSystem = parcello.FileSystem
