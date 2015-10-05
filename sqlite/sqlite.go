package sqlite

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	dbName string
	db     *sql.DB

	InsertStmt   *sql.Stmt
	QueryAllStmt *sql.Stmt
	DeleteStmt   *sql.Stmt

	rowCount int
}

func NewDB(dbName string) (*DB, error) {
	const (
		insertSQL = `
			INSERT INTO geo(uuid, created_at, updated_at, name, loc, borough, handicap, openallyear, lat, lng)
				VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`
		queryAllSQL = `
			SELECT uuid, created_at, updated_at, name, loc, borough, handicap, openallyear, lat, lng
				FROM geo;`
		deleteAllSQL = `
			DELETE FROM geo;`
		createSQL = `
			CREATE TABLE IF NOT EXISTS geo (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				uuid VARCHAR(40) NOT NULL,
				created_at INTEGER,
				updated_at INTEGER,
				name VARCHAR(100) NOT NULL,
				loc VARCHAR(200) NOT NULL,
				borough VARCHAR(50) NOT NULL,
				handicap BOOLEAN NOT NULL DEFAULT FALSE,
				openallyear BOOLEAN NOT NULL DEFAULT FALSE,
				lat DOUBLE PRECISION NOT NULL,
				lng DOUBLE PRECISION NOT NULL
			)`
	)

	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(createSQL); err != nil {
		return nil, err
	}

	istmt, err := db.Prepare(insertSQL)
	if err != nil {
		return nil, err
	}

	qstmt, err := db.Prepare(queryAllSQL)
	if err != nil {
		return nil, err
	}

	dstmt, err := db.Prepare(deleteAllSQL)
	if err != nil {
		return nil, err
	}

	return &DB{
		dbName:       dbName,
		db:           db,
		InsertStmt:   istmt,
		QueryAllStmt: qstmt,
		DeleteStmt:   dstmt,
	}, nil
}

func (db *DB) CountGeo() int {
	var count int
	if err := db.db.QueryRow("SELECT COUNT(*) FROM geo").Scan(&count); err != nil {
		return 0
	}
	return count
}

func (db *DB) ClearGeo() error {
	_, err := db.DeleteStmt.Exec()
	return err
}

func (db *DB) Close() error {
	if err := db.InsertStmt.Close(); err != nil {
		return err
	}

	if err := db.QueryAllStmt.Close(); err != nil {
		return err
	}

	return db.Close()
}
