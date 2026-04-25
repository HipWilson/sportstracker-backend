package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"github.com/user/sportstracker-backend/internal/database"
	"github.com/user/sportstracker-backend/internal/handlers"
)

func main() {
	// Load .env if present (local dev)
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Connect to database
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("could not run migrations: %v", err)
	}

	// Setup router
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// CORS — allows any origin so the frontend (GitHub Pages) can talk to this server
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Accept"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Handlers
	sh := handlers.NewSeriesHandler(db)
	rh := handlers.NewRatingHandler(db)

	// Serve uploaded images as static files
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// Serve Swagger UI
	r.Handle("/docs/*", http.StripPrefix("/docs/", http.FileServer(http.Dir("./docs"))))
	r.Get("/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/swagger.yaml")
	})

	// API routes
	r.Route("/series", func(r chi.Router) {
		r.Get("/", sh.ListSeries)
		r.Post("/", sh.CreateSeries)
		r.Get("/{id}", sh.GetSeries)
		r.Put("/{id}", sh.UpdateSeries)
		r.Delete("/{id}", sh.DeleteSeries)
		r.Post("/{id}/image", sh.UploadImage)

		// Rating sub-routes
		r.Post("/{id}/rating", rh.AddRating)
		r.Get("/{id}/rating", rh.GetRatings)
		r.Delete("/{id}/rating/{ratingId}", rh.DeleteRating)
	})

	fmt.Printf("🏆 SportsTracker API running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}