package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"testing"

	"forum/config"
	"forum/models"
	"forum/repository"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(config.CreateUserTable); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(config.CreatePostsTable); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(config.CreateImagesTable); err != nil {
		t.Fatal(err)
	}
	return db, func() { db.Close() }
}

func TestImageUpload(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewImageRepository(db)
	h := NewImageHandler(repo)
	ImageBaseDir = t.TempDir()

	// insert user and post
	db.Exec(`INSERT INTO user (user_id, username, email) VALUES ('u1','u','e')`)
	db.Exec(`INSERT INTO posts (post_id, user_id, title, content) VALUES ('p1','u1','t','c')`)

	// create sample image
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	buf := new(bytes.Buffer)
	png.Encode(buf, img)

	body := new(bytes.Buffer)
	mw := multipart.NewWriter(body)
	mw.WriteField("post_id", "p1")
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", `form-data; name="image"; filename="a.png"`)
	hdr.Set("Content-Type", "image/png")
	fw, _ := mw.CreatePart(hdr)
	io.Copy(fw, bytes.NewReader(buf.Bytes()))
	mw.Close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req = addUserContext(req, "u1")

	rr := httptest.NewRecorder()
	h.Upload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
	}

	files, _ := os.ReadDir(ImageBaseDir)
	if len(files) == 0 {
		t.Fatalf("no files saved")
	}

	imgRec, _ := repo.GetByPostID("p1")
	if imgRec == nil {
		t.Fatalf("no db record")
	}
}

// helper to provide user in context
func addUserContext(r *http.Request, id string) *http.Request {
	ctx := context.WithValue(r.Context(), "user", &models.User{ID: id})
	return r.WithContext(ctx)
}
