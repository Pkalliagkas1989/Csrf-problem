package repository

import (
	"database/sql"
	"time"

	"forum/models"
	"forum/utils"
)

type ImageRepository struct {
	db *sql.DB
}

func NewImageRepository(db *sql.DB) *ImageRepository {
	return &ImageRepository{db: db}
}

func (r *ImageRepository) Create(img models.Image) (*models.Image, error) {
	img.ID = utils.GenerateUUID()
	img.CreatedAt = time.Now()
	_, err := r.db.Exec(`INSERT INTO images (image_id, post_id, user_id, path, thumbnail_path, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		img.ID, img.PostID, img.UserID, img.Path, img.ThumbnailPath, img.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &img, nil
}

func (r *ImageRepository) GetByPostID(postID string) (*models.Image, error) {
	row := r.db.QueryRow(`SELECT image_id, post_id, user_id, path, thumbnail_path, created_at FROM images WHERE post_id = ? LIMIT 1`, postID)
	var img models.Image
	if err := row.Scan(&img.ID, &img.PostID, &img.UserID, &img.Path, &img.ThumbnailPath, &img.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &img, nil
}
