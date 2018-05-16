package sqldriver

import (
	"bytes"
	"errors"
	"strconv"
	"unicode"
)

// Allow digits and letters in bind params;  additionally runes are
// checked against underscores, meaning that bind params can have be
// alphanumeric with underscores.  Mind the difference between unicode
// digits and numbers, where '5' is a digit but '五' is not.
var allowedBindRunes = []*unicode.RangeTable{unicode.Letter, unicode.Digit}

// Bindvar types supported by Rebind, BindMap and BindStruct.
const (
	UNKNOWN = iota
	QUESTION
	DOLLAR
	NAMED
)

// BindType returns the bindtype for a given database given a drivername.
func BindType(driverName string) int {
	switch driverName {
	case "postgres", "pgx", "pq-timeouts", "cloudsqlpostgres":
		return DOLLAR
	case "mysql":
		return QUESTION
	case "sqlite3":
		return QUESTION
	case "oci8", "ora", "goracle":
		return NAMED
	}
	return UNKNOWN
}

// Rebind a query from the default bindtype (QUESTION) to the target bindtype.
func Rebind(bindType int, query string) string {
	if bindType != DOLLAR {
		return query
	}

	buffer := bytes.NewBuffer(make([]byte, 0, len(query)))
	position := 1

	for _, r := range query {
		if r == '?' {
			buffer.WriteRune('$')
			buffer.WriteString(strconv.Itoa(position))
			position++
		} else {
			buffer.WriteRune(r)
		}
	}

	return buffer.String()
}

// CompileNamedQuery - rebind a named query, returning a query and list of names
func CompileNamedQuery(qs string, bindType int) (query string, names []string, err error) {
	names = make([]string, 0, 10)
	rebound := bytes.Buffer{}

	inName := false
	last := len(qs) - 1
	currentVar := 1
	name := bytes.Buffer{}

	for i, b := range qs {
		// a ':' while we're in a name is an error
		if b == ':' {
			// if this is the second ':' in a '::' escape sequence, append a ':'
			if inName && i > 0 && qs[i-1] == ':' {
				rebound.WriteByte(':')
				inName = false
				continue
			} else if inName {
				err = errors.New("unexpected `:` while reading named param at " + strconv.Itoa(i))
				return query, names, err
			}
			inName = true
			name.Reset()
			// if we're in a name, and this is an allowed character, continue
		} else if inName && (unicode.IsOneOf(allowedBindRunes, rune(b)) || b == '_' || b == '.') && i != last {
			// append the byte to the name if we are in a name and not on the last byte
			name.WriteRune(b)
			// if we're in a name and it's not an allowed character, the name is done
		} else if inName {
			inName = false
			// if this is the final byte of the string and it is part of the name, then
			// make sure to add it to the name
			if i == last && unicode.IsOneOf(allowedBindRunes, rune(b)) {
				name.WriteRune(b)
			}
			// add the string representation to the names list
			names = append(names, name.String())
			// add a proper bindvar for the bindType
			switch bindType {
			// oracle only supports named type bind vars even for positional
			case NAMED:
				rebound.WriteByte(':')
				rebound.Write(name.Bytes())
			case QUESTION, UNKNOWN:
				rebound.WriteByte('?')
			case DOLLAR:
				rebound.WriteByte('$')
				for _, b := range strconv.Itoa(currentVar) {
					rebound.WriteRune(b)
				}
				currentVar++
			}
			// add this byte to string unless it was not part of the name
			if i != last {
				rebound.WriteRune(b)
			} else if !unicode.IsOneOf(allowedBindRunes, rune(b)) {
				rebound.WriteRune(b)
			}
		} else {
			// this is a normal byte and should just go onto the rebound query
			rebound.WriteRune(b)
		}
	}

	return rebound.String(), names, err
}
