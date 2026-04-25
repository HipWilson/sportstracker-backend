package models

import "time"

// Series represents a sports documentary or series being tracked.
type Series struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Sport       string     `json:"sport"`
	Platform    string     `json:"platform"`
	Status      string     `json:"status"`
	Episodes    int        `json:"episodes"`
	Year        *int       `json:"year"`
	Description string     `json:"description"`
	ImageURL    string     `json:"image_url"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	AvgRating   *float64   `json:"avg_rating,omitempty"`
	RatingCount int        `json:"rating_count,omitempty"`
}

// SeriesInput is used for create/update requests.
type SeriesInput struct {
	Title       string `json:"title"`
	Sport       string `json:"sport"`
	Platform    string `json:"platform"`
	Status      string `json:"status"`
	Episodes    int    `json:"episodes"`
	Year        *int   `json:"year"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
}

// Rating represents a user rating for a series.
type Rating struct {
	ID        int       `json:"id"`
	SeriesID  int       `json:"series_id"`
	Score     int       `json:"score"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

// RatingInput is used for rating creation.
type RatingInput struct {
	Score   int    `json:"score"`
	Comment string `json:"comment"`
}

// PaginatedSeries wraps a list of series with pagination metadata.
type PaginatedSeries struct {
	Data       []Series `json:"data"`
	Total      int      `json:"total"`
	Page       int      `json:"page"`
	Limit      int      `json:"limit"`
	TotalPages int      `json:"total_pages"`
}

// ErrorResponse is a standard JSON error envelope.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}