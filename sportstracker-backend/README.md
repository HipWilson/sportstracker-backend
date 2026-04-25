# 🏆 SportsTracker — Backend

API REST construida en Go con PostgreSQL para rastrear series y documentales deportivos.
El servidor solo responde JSON, nunca genera HTML.

**🖥️ Frontend repo:** [sportstracker-frontend](https://github.com/TU_USUARIO/sportstracker-frontend)  
**🌐 API en producción:** https://TU_APP.up.railway.app  
**📄 Swagger UI:** https://TU_APP.up.railway.app/docs/

---

## Stack

| Tecnología | Uso |
|---|---|
| Go 1.21 | Lenguaje del servidor |
| chi v5 | Router HTTP |
| PostgreSQL | Base de datos |
| go-chi/cors | Middleware CORS |
| Railway | Deploy en producción |

---

## ¿Qué es CORS y qué se configuró?

CORS (Cross-Origin Resource Sharing) es un mecanismo de seguridad del navegador que bloquea peticiones HTTP entre orígenes distintos (diferente dominio, puerto o protocolo). Como el frontend corre en GitHub Pages y el backend en Railway, el navegador los trata como orígenes distintos y bloquearía las peticiones `fetch()` por defecto.

Se configuró el middleware `go-chi/cors` con los siguientes headers:

```
Access-Control-Allow-Origin:  *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

---

## Endpoints de la API

### Series

| Método | Ruta | Descripción | Código éxito |
|---|---|---|---|
| GET | `/series` | Listar series (con filtros) | 200 |
| POST | `/series` | Crear una serie nueva | 201 |
| GET | `/series/:id` | Obtener una serie por ID | 200 |
| PUT | `/series/:id` | Editar una serie existente | 200 |
| DELETE | `/series/:id` | Eliminar una serie | 204 |
| POST | `/series/:id/image` | Subir imagen de portada (máx 1MB) | 200 |

### Ratings

| Método | Ruta | Descripción | Código éxito |
|---|---|---|---|
| GET | `/series/:id/rating` | Obtener calificaciones de una serie | 200 |
| POST | `/series/:id/rating` | Agregar una calificación | 201 |
| DELETE | `/series/:id/rating/:ratingId` | Eliminar una calificación | 204 |

### Documentación

| Método | Ruta | Descripción |
|---|---|---|
| GET | `/docs/` | Swagger UI interactivo |
| GET | `/swagger.yaml` | Spec OpenAPI 3.0 en YAML |

---

## Query params en GET /series

| Parámetro | Descripción | Ejemplo |
|---|---|---|
| `q` | Búsqueda por título, descripción o deporte | `?q=formula` |
| `page` | Número de página (default: 1) | `?page=2` |
| `limit` | Resultados por página, máx 100 (default: 20) | `?limit=10` |
| `sort` | Campo para ordenar: `title`, `sport`, `year`, `created_at`, `rating` | `?sort=title` |
| `order` | Dirección: `ASC` o `DESC` (default: DESC) | `?order=ASC` |
| `sport` | Filtrar por deporte específico | `?sport=futbol` |
| `status` | Filtrar por estado: `pending`, `watching`, `completed`, `dropped` | `?status=watching` |

Ejemplo completo:
```
GET /series?q=documental&sport=futbol&sort=rating&order=DESC&page=1&limit=10
```

---

## Códigos HTTP utilizados

| Código | Cuándo se usa |
|---|---|
| 200 OK | Consulta o actualización exitosa |
| 201 Created | Recurso creado exitosamente |
| 204 No Content | Eliminación exitosa |
| 400 Bad Request | JSON inválido o campos que no pasan validación |
| 404 Not Found | El recurso solicitado no existe |
| 500 Internal Server Error | Error inesperado del servidor |

---

## Estructura del proyecto

```
sportstracker-backend/
├── cmd/
│   └── server/
│       └── main.go          # Entrada: servidor HTTP, router, CORS
├── internal/
│   ├── database/
│   │   └── db.go            # Conexión a PostgreSQL y migraciones automáticas
│   ├── handlers/
│   │   ├── series.go        # CRUD de series, paginación, búsqueda, imagen
│   │   └── rating.go        # Endpoints del sistema de calificaciones
│   └── models/
│       └── models.go        # Structs: Series, Rating, inputs y respuestas
├── docs/
│   ├── swagger.yaml         # Especificación OpenAPI 3.0
│   └── index.html           # Swagger UI servido desde el servidor
├── go.mod
├── go.sum
├── .env.example
└── .gitignore
```

---

## Correr localmente

### Requisitos
- Go 1.21 o superior
- PostgreSQL corriendo localmente

### Pasos

```bash
# 1. Clonar el repositorio
git clone https://github.com/TU_USUARIO/sportstracker-backend.git
cd sportstracker-backend

# 2. Copiar variables de entorno
cp .env.example .env
```

Editar `.env` con tus credenciales:

```env
PORT=8080
DATABASE_URL=postgres://usuario:contrasena@localhost:5432/sportstracker?sslmode=disable
BASE_URL=http://localhost:8080
```

```bash
# 3. Crear la base de datos en PostgreSQL
psql -U postgres -c "CREATE DATABASE sportstracker;"

# 4. Descargar dependencias
go mod download

# 5. Correr el servidor
go run ./cmd/server
```

Las tablas `series` y `ratings` se crean automáticamente al iniciar si no existen.

El servidor queda disponible en `http://localhost:8080`.  
La documentación Swagger en `http://localhost:8080/docs/`.

---

## Deploy en Railway

1. Crear cuenta en [railway.app](https://railway.app)
2. **New Project** → Deploy from GitHub → seleccionar `sportstracker-backend`
3. **Add Plugin** → PostgreSQL (Railway provee `DATABASE_URL` automáticamente)
4. En **Variables**, agregar:
   ```
   BASE_URL = https://tu-app.up.railway.app
   ```
5. En **Settings → Deploy → Start Command**:
   ```
   go run ./cmd/server
   ```
6. Railway asignará una URL pública automáticamente.

---

## Ejemplo de peticiones

```bash
# Listar series
curl https://TU_APP.up.railway.app/series

# Crear serie
curl -X POST https://TU_APP.up.railway.app/series \
  -H "Content-Type: application/json" \
  -d '{"title":"Drive to Survive","sport":"Formula 1","status":"watching","episodes":10,"year":2019}'

# Buscar
curl "https://TU_APP.up.railway.app/series?q=formula&sort=rating&order=DESC"

# Agregar calificación
curl -X POST https://TU_APP.up.railway.app/series/1/rating \
  -H "Content-Type: application/json" \
  -d '{"score":9,"comment":"Excelente documental"}'
```