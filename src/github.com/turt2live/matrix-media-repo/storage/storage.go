package storage

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq" // postgres driver
	"github.com/turt2live/matrix-media-repo/storage/schema"
	"github.com/turt2live/matrix-media-repo/types"
)

const selectMedia = "SELECT origin, media_id, upload_name, content_type, user_id, sha256_hash, size_bytes, location, creation_ts FROM media WHERE origin = $1 and media_id = $2;"
const selectMediaByHash = "SELECT origin, media_id, upload_name, content_type, user_id, sha256_hash, size_bytes, location, creation_ts FROM media WHERE sha256_hash = $1;"
const insertMedia = "INSERT INTO media (origin, media_id, upload_name, content_type, user_id, sha256_hash, size_bytes, location, creation_ts) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);"
const selectSizeOfFolder = "SELECT COALESCE(SUM(size_bytes), 0) AS size_total FROM media WHERE location ILIKE $1 || '%'"

type folderSize struct {
	Size int64
}

type Database struct {
	db         *sql.DB
	statements statements
}

type statements struct {
	selectMedia *sql.Stmt
	selectMediaByHash *sql.Stmt
	insertMedia *sql.Stmt
	selectSizeOfFolder *sql.Stmt
}

func OpenDatabase(connectionString string) (*Database, error) {
	var d Database
	var err error

	if d.db, err = sql.Open("postgres", connectionString); err != nil {
		return nil, err
	}

	schema.PrepareMedia(d.db)

	// prepare a bunch of statements for use
	if d.statements.selectMedia, err = d.db.Prepare(selectMedia); err != nil { return nil, err }
	if d.statements.selectMediaByHash, err = d.db.Prepare(selectMediaByHash); err != nil { return nil, err }
	if d.statements.insertMedia, err = d.db.Prepare(insertMedia); err != nil { return nil, err }
	if d.statements.selectSizeOfFolder, err = d.db.Prepare(selectSizeOfFolder); err != nil { return nil, err }

	return &d, nil
}

func (d *Database) InsertMedia(ctx context.Context, media *types.Media) error {
	_, err := d.statements.insertMedia.ExecContext(
		ctx,
		media.Origin,
		media.MediaId,
		media.UploadName,
		media.ContentType,
		media.UserId,
		media.Sha256Hash,
		media.SizeBytes,
		media.Location,
		media.CreationTs,
	)

	return err
}

func (d *Database) GetMediaByHash(ctx context.Context, hash string) ([]types.Media, error) {
	rows, err := d.statements.selectMediaByHash.QueryContext(ctx, hash)
	if err != nil {
		return nil, err
	}

	var results []types.Media
	for rows.Next() {
		obj := types.Media{}
		err = rows.Scan(
			&obj.Origin,
			&obj.MediaId,
			&obj.UploadName,
			&obj.ContentType,
			&obj.UserId,
			&obj.Sha256Hash,
			&obj.SizeBytes,
			&obj.Location,
			&obj.CreationTs,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, obj)
	}

	return results, nil
}

func (d *Database) GetSizeOfFolderBytes(ctx context.Context, folderPath string) (int64, error) {
	r := &folderSize{}
	err := d.statements.selectSizeOfFolder.QueryRowContext(ctx, folderPath).Scan(&r.Size)
	return r.Size, err
}