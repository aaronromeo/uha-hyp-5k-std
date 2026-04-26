package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

//go:embed templates/*.html
var templateFS embed.FS

func main() {
	dataDir := "data"
	if d := os.Getenv("DATA_DIR"); d != "" {
		dataDir = d
	}
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	program, err := loadProgram(dataDir + "/program.json")
	if err != nil {
		log.Fatalf("failed to load program.json: %v", err)
	}

	prs, err := loadPRs(dataDir + "/prs.json")
	if err != nil {
		log.Fatalf("failed to load prs.json: %v", err)
	}

	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	app := &App{
		program:   program,
		prs:       prs,
		templates: tmpl,
	}

	http.HandleFunc("/", app.handleIndex)
	http.HandleFunc("/workout", app.handleWorkout)

	fmt.Printf("Listening on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func loadProgram(path string) (Program, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Program
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return p, nil
}

func loadPRs(path string) (PRs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var prs PRs
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, err
	}
	return prs, nil
}
