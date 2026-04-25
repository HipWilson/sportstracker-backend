package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/user/sportstracker-backend/internal/models"
)

// RatingHandler holds the DB reference for rating operations.
type RatingHandler struct {
	db *sql.DB
}

// NewRatingHandler creates a new handler.
func NewRatingHandler(db *sql.DB) *RatingHandler {
	return &RatingHandler{db: db}
}

// AddRating handles POST /series/:id/rating
func (h *RatingHandler) AddRating(w http.ResponseWriter, r *http.Request) {
	seriesID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid series id")
		return
	}

	var exists bool
	h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM series WHERE id=$1)", seriesID).Scan(&exists)
	if !exists {
		writeError(w, http.StatusNotFound, "series not found")
		return
	}

	var input models.RatingInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	errs := map[string]string{}
	if input.Score < 1 || input.Score > 10 {
		errs["score"] = "score must be between 1 and 10"
	}
	if len(errs) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.ErrorResponse{Error: "validation failed", Details: errs})
		return
	}

	var rating models.Rating
	err = h.db.QueryRow(`
		INSERT INTO ratings (series_id, score, comment)
		VALUES ($1, $2, $3)
		RETURNING id, series_id, score, comment, created_at`,
		seriesID, input.Score, input.Comment,
	).Scan(&rating.ID, &rating.SeriesID, &rating.Score, &rating.Comment, &rating.CreatedAt)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save rating")
		return
	}

	writeJSON(w, http.StatusCreated, rating)
}

// GetRatings handles GET /series/:id/rating
func (h *RatingHandler) GetRatings(w http.ResponseWriter, r *http.Request) {
	seriesID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid series id")
		return
	}

	var exists bool
	h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM series WHERE id=$1)", seriesID).Scan(&exists)
	if !exists {
		writeError(w, http.StatusNotFound, "series not found")
		return
	}

	rows, err := h.db.Query(`
		SELECT id, series_id, score, comment, created_at
		FROM ratings WHERE series_id=$1
		ORDER BY created_at DESC`, seriesID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	var ratings []models.Rating
	var totalScore int
	for rows.Next() {
		var rat models.Rating
		if err := rows.Scan(&rat.ID, &rat.SeriesID, &rat.Score, &rat.Comment, &rat.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		ratings = append(ratings, rat)
		totalScore += rat.Score
	}

	var avg *float64
	if len(ratings) > 0 {
		v := float64(totalScore) / float64(len(ratings))
		avg = &v
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ratings":   ratings,
		"count":     len(ratings),
		"avg_score": avg,
	})
}

// DeleteRating handles DELETE /series/:id/rating/:ratingId
func (h *RatingHandler) DeleteRating(w http.ResponseWriter, r *http.Request) {
	seriesID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid series id")
		return
	}
	ratingID, err := strconv.Atoi(chi.URLParam(r, "ratingId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rating id")
		return
	}

	result, err := h.db.Exec("DELETE FROM ratings WHERE id=$1 AND series_id=$2", ratingID, seriesID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		writeError(w, http.StatusNotFound, "rating not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}