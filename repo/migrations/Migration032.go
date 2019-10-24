package migrations

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	_ "github.com/mutecomm/go-sqlcipher"
)

const (
	// migrationAlterMessagesAM09Err alters the table to add the err column.
	migrationAlterMessagesAM09Err = "alter table messages add err text;"
	// migrationAlterMessagesAM09ReceivedAt alters the table to add the received_at column.
	migrationAlterMessagesAM09ReceivedAt = "alter table messages add received_at integer;"
	// MigrationCreateMessagesAM09MessagesCreateSQLDown the messages create sql
	MigrationCreateMessagesAM09MessagesCreateSQLDown = "create table messages (messageID text primary key not null, orderID text, message_type integer, message blob, peerID text, url text, acknowledged bool, tries integer, created_at integer, updated_at integer);"
	// migrationRenameMessagesAM09MessagesCreateSQL the messages create sql
	migrationRenameMessagesAM09MessagesCreateSQL = "ALTER TABLE messages RENAME TO temp_messages;"
	// migrationInsertMessagesAM09Messages the messages create sql
	migrationInsertMessagesAM09Messages = "INSERT INTO messages SELECT messageID, orderID, message_type, message, peerID, url, acknowledged, tries, created_at, updated_at FROM temp_messages;"
	// migrationCreateMessagesAM09MessagesDeleteSQL the messages delete sql
	migrationCreateMessagesAM09MessagesDeleteSQL = "drop table if exists temp_messages;"
	// migrationCreateMessagesAM09UpVer set the repo Up version
	migrationCreateMessagesAM09UpVer = "33"
	// migrationCreateMessagesAM09DownVer set the repo Down version
	migrationCreateMessagesAM09DownVer = "32"
)

// Migration032  migration struct
type Migration032 struct{}

// Up the migration Up code
func (Migration032) Up(repoPath, databasePassword string, testnetEnabled bool) error {
	var (
		databaseFilePath    string
		repoVersionFilePath = path.Join(repoPath, "repover")
	)
	if testnetEnabled {
		databaseFilePath = path.Join(repoPath, "datastore", "testnet.db")
	} else {
		databaseFilePath = path.Join(repoPath, "datastore", "mainnet.db")
	}

	db, err := sql.Open("sqlite3", databaseFilePath)
	if err != nil {
		return err
	}
	defer db.Close()
	if databasePassword != "" {
		p := fmt.Sprintf("pragma key = '%s';", databasePassword)
		_, err := db.Exec(p)
		if err != nil {
			return err
		}
	}

	upSequence := strings.Join([]string{
		migrationAlterMessagesAM09Err,
		migrationAlterMessagesAM09ReceivedAt,
	}, " ")

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(upSequence); err != nil {
		if rErr := tx.Rollback(); rErr != nil {
			return fmt.Errorf("rollback failed: (%s) due to (%s)", rErr.Error(), err.Error())
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	// Bump schema version
	err = ioutil.WriteFile(repoVersionFilePath, []byte(migrationCreateMessagesAM09UpVer), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

// Down the migration Down code
func (Migration032) Down(repoPath, databasePassword string, testnetEnabled bool) error {
	var (
		databaseFilePath    string
		repoVersionFilePath = path.Join(repoPath, "repover")
	)
	if testnetEnabled {
		databaseFilePath = path.Join(repoPath, "datastore", "testnet.db")
	} else {
		databaseFilePath = path.Join(repoPath, "datastore", "mainnet.db")
	}

	db, err := sql.Open("sqlite3", databaseFilePath)
	if err != nil {
		return err
	}
	defer db.Close()
	if databasePassword != "" {
		p := fmt.Sprintf("pragma key = '%s';", databasePassword)
		_, err := db.Exec(p)
		if err != nil {
			return err
		}
	}

	downSequence := strings.Join([]string{
		migrationRenameMessagesAM09MessagesCreateSQL,
		MigrationCreateMessagesAM09MessagesCreateSQLDown,
		migrationInsertMessagesAM09Messages,
		migrationCreateMessagesAM09MessagesDeleteSQL,
	}, " ")

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(downSequence); err != nil {
		if rErr := tx.Rollback(); rErr != nil {
			return fmt.Errorf("rollback failed: (%s) due to (%s)", rErr.Error(), err.Error())
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	// Bump schema version
	err = ioutil.WriteFile(repoVersionFilePath, []byte(migrationCreateMessagesAM09DownVer), os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
