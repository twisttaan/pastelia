package handler
import (
	"context"
	"encoding/json"
	"net/http"
	"os"
  "fmt"
  "math/rand"
	"strings"
	"time"
	"github.com/jackc/pgx/v4/pgxpool"
)

func RandomSlug() string {
	consonants := "bcdfghjklmnpqrstvwxyz"
	vowels := "aeiou"

	rand.Seed(time.Now().UnixNano())

	var slug strings.Builder
	for i := 0; i < 5; i++ {
		slug.WriteByte(consonants[rand.Intn(21)])
		slug.WriteByte(vowels[rand.Intn(5)])
	}

	return slug.String()
}

type Snippet struct {
	Code string `json:"code"`
	Lang string `json:"lang"`
}

func (snippet *Snippet) String() string {
	s, _ := json.Marshal(snippet)
	return string(s)
}

func CreateSnippet(dbpool *pgxpool.Pool, snippet *Snippet) (string, error) {
	slug := RandomSlug()
	_, err := dbpool.Exec(
		context.Background(),
		"insert into snippets (slug, code, lang) values ($1, $2, $3)",
		slug,
		snippet.Code,
		snippet.Lang,
	)

	if err != nil {
		return "", err
	}

	return slug, nil
}

func ReadSnippet(dbpool *pgxpool.Pool, slug string) (*Snippet, error) {
	var code string
	var lang string

	err := dbpool.
		QueryRow(
			context.Background(),
			"select code, lang from snippets where slug=$1",
			slug,
		).
		Scan(&code, &lang)

	snippet := &Snippet{Code: code, Lang: lang}
	if err != nil {
		return snippet, err
	}

	return snippet, nil
}


func Index(w http.ResponseWriter, r *http.Request) {
  databaseUrl := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
    os.Getenv("POSTGRES_HOST"),
    os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"),
	)

	dbpool, err := pgxpool.Connect(context.Background(), databaseUrl)
	if err != nil {
		panic("Unable to connect to database")
	}

  switch r.Method {
  case http.MethodGet:
    qs := r.URL.Query()
    if !qs.Has("q") {
      http.NotFound(w, r)
      return
    }

    snippet, err := ReadSnippet(dbpool, qs.Get("q"))
    if err != nil {
      http.NotFound(w, r)
      return
    }

    w.Header().Add("Content-Type", "application/json")
    w.Write([]byte(snippet.String()))
  case http.MethodPost:
    var snippet Snippet
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&snippet)
    if err != nil || snippet.Lang == "" {
      w.WriteHeader(http.StatusUnprocessableEntity)
      return
    }

    slug, err := CreateSnippet(dbpool, &snippet)
    if err != nil {
      w.WriteHeader(http.StatusInternalServerError)
    }

    w.Write([]byte(slug))
  default:
    w.WriteHeader(http.StatusMethodNotAllowed)
  }
}