package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/user/sportstracker-backend/internal/models"
)

// SeriesHandler holds the DB reference for series operations.
type SeriesHandler struct {
	db *sql.DB
}

// NewSeriesHandler creates a new handler.
func NewSeriesHandler(db *sql.DB) *SeriesHandler {
	return &SeriesHandler{db: db}
}

// writeJSON sends a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError sends a standard error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, models.ErrorResponse{Error: msg})
}

// validateSeriesInput checks required fields and returns field-level errors.
func validateSeriesInput(input models.SeriesInput) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(input.Title) == "" {
		errs["title"] = "title is required"
	} else if len(input.Title) > 200 {
		errs["title"] = "title must be 200 characters or fewer"
	}
	if strings.TrimSpace(input.Sport) == "" {
		errs["sport"] = "sport is required"
	}
	validStatuses := map[string]bool{"pending": true, "watching": true, "completed": true, "dropped": true}
	if input.Status != "" && !validStatuses[input.Status] {
		errs["status"] = "status must be one of: pending, watching, completed, dropped"
	}
	if input.Episodes < 0 {
		errs["episodes"] = "episodes cannot be negative"
	}
	if input.Year != nil && (*input.Year < 1900 || *input.Year > 2100) {
		errs["year"] = "year must be between 1900 and 2100"
	}
	return errs
}

// ListSeries handles GET /series with optional ?q=, ?page=, ?limit=, ?sort=, ?order=, ?sport=, ?status=
func (h *SeriesHandler) ListSeries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	search := strings.TrimSpace(q.Get("q"))
	sport := strings.TrimSpace(q.Get("sport"))
	status := strings.TrimSpace(q.Get("status"))

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	sortField := q.Get("sort")
	allowedSorts := map[string]string{
		"title": "s.title", "sport": "s.sport", "year": "s.year",
		"created_at": "s.created_at", "rating": "avg_rating",
	}
	orderBy, ok := allowedSorts[sortField]
	if !ok {
		orderBy = "s.created_at"
	}

	orderDir := strings.ToUpper(q.Get("order"))
	if orderDir != "ASC" && orderDir != "DESC" {
		orderDir = "DESC"
	}

	// Build WHERE conditions
	conditions := []string{}
	args := []any{}
	idx := 1

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(s.title ILIKE $%d OR s.description ILIKE $%d OR s.sport ILIKE $%d)", idx, idx+1, idx+2))
		like := "%" + search + "%"
		args = append(args, like, like, like)
		idx += 3
	}
	if sport != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(s.sport) = LOWER($%d)", idx))
		args = append(args, sport)
		idx++
	}
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("s.status = $%d", idx))
		args = append(args, status)
		idx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM series s %s`, where)
	var total int
	if err := h.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	offset := (page - 1) * limit

	// Main query
	mainQuery := fmt.Sprintf(`
		SELECT s.id, s.title, s.sport, s.platform, s.status, s.episodes,
		       s.year, s.description, s.image_url, s.created_at, s.updated_at,
		       ROUND(AVG(r.score)::numeric, 2) AS avg_rating,
		       COUNT(r.id) AS rating_count
		FROM series s
		LEFT JOIN ratings r ON r.series_id = s.id
		%s
		GROUP BY s.id
		ORDER BY %s %s NULLS LAST
		LIMIT $%d OFFSET $%d
	`, where, orderBy, orderDir, idx, idx+1)

	args = append(args, limit, offset)

	rows, err := h.db.Query(mainQuery, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	series := []models.Series{}
	for rows.Next() {
		var s models.Series
		var year sql.NullInt64
		var avgRating sql.NullFloat64
		var ratingCount int
		err := rows.Scan(&s.ID, &s.Title, &s.Sport, &s.Platform, &s.Status,
			&s.Episodes, &year, &s.Description, &s.ImageURL,
			&s.CreatedAt, &s.UpdatedAt, &avgRating, &ratingCount)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		if year.Valid {
			y := int(year.Int64)
			s.Year = &y
		}
		if avgRating.Valid {
			v := avgRating.Float64
			s.AvgRating = &v
		}
		s.RatingCount = ratingCount
		series = append(series, s)
	}

	writeJSON(w, http.StatusOK, models.PaginatedSeries{
		Data:       series,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// GetSeries handles GET /series/:id
func (h *SeriesHandler) GetSeries(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	query := `
		SELECT s.id, s.title, s.sport, s.platform, s.status, s.episodes,
		       s.year, s.description, s.image_url, s.created_at, s.updated_at,
		       ROUND(AVG(r.score)::numeric, 2), COUNT(r.id)
		FROM series s
		LEFT JOIN ratings r ON r.series_id = s.id
		WHERE s.id = $1
		GROUP BY s.id
	`

	var s models.Series
	var year sql.NullInt64
	var avgRating sql.NullFloat64
	var ratingCount int

	err = h.db.QueryRow(query, id).Scan(
		&s.ID, &s.Title, &s.Sport, &s.Platform, &s.Status,
		&s.Episodes, &year, &s.Description, &s.ImageURL,
		&s.CreatedAt, &s.UpdatedAt, &avgRating, &ratingCount,
	)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "series not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	if year.Valid {
		y := int(year.Int64)
		s.Year = &y
	}
	if avgRating.Valid {
		v := avgRating.Float64
		s.AvgRating = &v
	}
	s.RatingCount = ratingCount

	writeJSON(w, http.StatusOK, s)
}

// CreateSeries handles POST /series
func (h *SeriesHandler) CreateSeries(w http.ResponseWriter, r *http.Request) {
	var input models.SeriesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if input.Status == "" {
		input.Status = "pending"
	}

	if errs := validateSeriesInput(input); len(errs) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "validation failed", Details: errs})
		return
	}

	var id int
	err := h.db.QueryRow(`
		INSERT INTO series (title, sport, platform, status, episodes, year, description, image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		input.Title, input.Sport, input.Platform, input.Status,
		input.Episodes, input.Year, input.Description, input.ImageURL,
	).Scan(&id)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create series")
		return
	}

	// Return the full created object
	h.respondWithSeries(w, http.StatusCreated, id)
}

// UpdateSeries handles PUT /series/:id
func (h *SeriesHandler) UpdateSeries(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Check it exists
	var exists bool
	h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM series WHERE id=$1)", id).Scan(&exists)
	if !exists {
		writeError(w, http.StatusNotFound, "series not found")
		return
	}

	var input models.SeriesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if errs := validateSeriesInput(input); len(errs) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "validation failed", Details: errs})
		return
	}

	_, err = h.db.Exec(`
		UPDATE series SET title=$1, sport=$2, platform=$3, status=$4,
		episodes=$5, year=$6, description=$7, image_url=$8, updated_at=NOW()
		WHERE id=$9`,
		input.Title, input.Sport, input.Platform, input.Status,
		input.Episodes, input.Year, input.Description, input.ImageURL, id,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update series")
		return
	}

	h.respondWithSeries(w, http.StatusOK, id)
}

// DeleteSeries handles DELETE /series/:id
func (h *SeriesHandler) DeleteSeries(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	result, err := h.db.Exec("DELETE FROM series WHERE id=$1", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete series")
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "series not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UploadImage handles POST /series/:id/image (multipart form, field "image")
func (h *SeriesHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var exists bool
	h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM series WHERE id=$1)", id).Scan(&exists)
	if !exists {
		writeError(w, http.StatusNotFound, "series not found")
		return
	}

	// Limit upload size to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "image must be 1MB or smaller")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "field 'image' is required")
		return
	}
	defer file.Close()

	// Validate content type
	allowedTypes := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
		"image/gif":  ".gif",
	}
	contentType := header.Header.Get("Content-Type")
	ext, ok := allowedTypes[contentType]
	if !ok {
		ext = filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".jpg"
		}
	}

	// Save to ./uploads/
	if err := os.MkdirAll("./uploads", 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "could not create uploads directory")
		return
	}

	filename := fmt.Sprintf("%d_%d%s", id, time.Now().UnixNano(), ext)
	dst, err := os.Create("./uploads/" + filename)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save image")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		writeError(w, http.StatusInternalServerError, "could not write image")
		return
	}

	// Build URL — use BASE_URL env or a relative path
	baseURL := os.Getenv("BASE_URL")
	imageURL := fmt.Sprintf("%s/uploads/%s", baseURL, filename)

	_, err = h.db.Exec("UPDATE series SET image_url=$1, updated_at=NOW() WHERE id=$2", imageURL, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not update image URL")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"image_url": imageURL})
}

// respondWithSeries is a helper to return a full series object after create/update.
func (h *SeriesHandler) respondWithSeries(w http.ResponseWriter, status int, id int) {
	var s models.Series
	var year sql.NullInt64
	var avgRating sql.NullFloat64
	var ratingCount int

	err := h.db.QueryRow(`
		SELECT s.id, s.title, s.sport, s.platform, s.status, s.episodes,
		       s.year, s.description, s.image_url, s.created_at, s.updated_at,
		       ROUND(AVG(r.score)::numeric, 2), COUNT(r.id)
		FROM series s
		LEFT JOIN ratings r ON r.series_id = s.id
		WHERE s.id = $1
		GROUP BY s.id`, id).Scan(
		&s.ID, &s.Title, &s.Sport, &s.Platform, &s.Status,
		&s.Episodes, &year, &s.Description, &s.ImageURL,
		&s.CreatedAt, &s.UpdatedAt, &avgRating, &ratingCount,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not retrieve series")
		return
	}
	if year.Valid {
		y := int(year.Int64)
		s.Year = &y
	}
	if avgRating.Valid {
		v := avgRating.Float64
		s.AvgRating = &v
	}
	s.RatingCount = ratingCount
	writeJSON(w, status, s)
}