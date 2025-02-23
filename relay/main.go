package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fiatjaf/eventstore/postgresql"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/policies"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	fmt.Print(`
	LUMINA RELAY
`)

	// create the relay instance
	relay := khatru.NewRelay()

	// set up relay properties with environment variable configuration
	relay.Info.Name = getEnv("RELAY_NAME", "LUMINA Relay")
	relay.Info.PubKey = getEnv("RELAY_PUBKEY", "79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798")
	relay.Info.Description = getEnv("RELAY_DESCRIPTION", "LUMINA Relay")
	relay.Info.Icon = getEnv("RELAY_ICON", "https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Fliquipedia.net%2Fcommons%2Fimages%2F3%2F35%2FSCProbe.jpg&f=1&nofb=1&ipt=0cbbfef25bce41da63d910e86c3c343e6c3b9d63194ca9755351bb7c2efa3359&ipo=images")

	// Print relay information
	fmt.Printf("Name: %s\n", relay.Info.Name)
	fmt.Printf("Public Key: %s\n", relay.Info.PubKey)
	fmt.Printf("Description: %s\n\n", relay.Info.Description)

	// Configure PostgreSQL connection with environment variable
	postgresURL := getEnv("POSTGRES_URL", "postgres://postgres:postgres@postgres/postgres?sslmode=disable")
	db := postgresql.PostgresBackend{DatabaseURL: postgresURL}
	if err := db.Init(); err != nil {
		panic(err)
	}

	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.DeleteEvent = append(relay.DeleteEvent, db.DeleteEvent)
	relay.ReplaceEvent = append(relay.ReplaceEvent, db.ReplaceEvent)
	relay.CountEvents = append(relay.CountEvents, db.CountEvents)

	relay.RejectEvent = append(
		relay.RejectEvent,
		policies.PreventLargeTags(120),
		policies.PreventTimestampsInThePast(time.Hour*2),
		policies.PreventTimestampsInTheFuture(time.Minute*30),
	)

	mux := relay.Router()
	// set up other http handlers
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html")

		// Query the total number of events
		count := 0
		row := db.DB.QueryRow("SELECT COUNT(*) FROM event")
		if err := row.Scan(&count); err != nil {
			fmt.Printf("Error counting events: %v\n", err)
		}

		// Improved HTML content with link to stats page
		fmt.Fprintf(w, `
			<!DOCTYPE html>
			<html lang="en">
			<head>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<title>Scrapestr Relay</title>
				<style>
					body {
						font-family: Arial, sans-serif;
						background-color: #f4f4f4;
						margin: 0;
						padding: 0;
						display: flex;
						justify-content: center;
						align-items: center;
						height: 100vh;
					}
					.container {
						background-color: #fff;
						padding: 20px;
						border-radius: 8px;
						box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
						text-align: center;
					}
					h1 {
						color: #333;
					}
					p {
						color: #666;
						margin: 10px 0;
					}
					a {
						color: #007bff;
						text-decoration: none;
					}
					a:hover {
						text-decoration: underline;
					}
				</style>
			</head>
			<body>
				<div class="container">
					<h1>Welcome to LUMINA Relay!</h1>
					<p>Number of events stored: %d</p>
					<p><a href="/stats">View Event Stats</a></p>
				</div>
			</body>
			</html>
		`, count)
	})

	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/html")

		// Query the number of events for each kind, sorted by kind
		rows, err := db.DB.Query("SELECT kind, COUNT(*) FROM event GROUP BY kind ORDER BY kind")
		if err != nil {
			fmt.Printf("Error querying event kinds: %v\n", err)
			return
		}
		defer rows.Close()

		stats := make(map[string]int)
		for rows.Next() {
			var kind string
			var count int
			if err := rows.Scan(&kind, &count); err != nil {
				fmt.Printf("Error scanning row: %v\n", err)
				return
			}
			stats[kind] = count
		}

		// Improved HTML content for stats
		fmt.Fprintf(w, `
			<!DOCTYPE html>
			<html lang="en">
			<head>
				<meta charset="UTF-8">
				<meta name="viewport" content="width=device-width, initial-scale=1.0">
				<title>Scrapestr Relay Stats</title>
				<style>
					body {
						font-family: Arial, sans-serif;
						background-color: #f4f4f4;
						margin: 0;
						padding: 0;
						display: flex;
						justify-content: center;
						align-items: center;
						height: 100vh;
					}
					.container {
						background-color: #fff;
						padding: 20px;
						border-radius: 8px;
						box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
						text-align: center;
					}
					h1 {
						color: #333;
					}
					table {
						width: 100%%;
						border-collapse: collapse;
					}
					th, td {
						padding: 10px;
						border: 1px solid #ddd;
					}
					th {
						background-color: #f4f4f4;
					}
				</style>
			</head>
			<body>
				<div class="container">
					<h1>Event Stats</h1>
					<table>
						<tr>
							<th>Kind</th>
							<th>Count</th>
						</tr>
		`)
		for kind, count := range stats {
			fmt.Fprintf(w, `
						<tr>
							<td>%s</td>
							<td>%d</td>
						</tr>
			`, kind, count)
		}
		fmt.Fprintf(w, `
					</table>
				</div>
			</body>
			</html>
		`)
	})

	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Query the number of events for each kind, sorted by kind
		rows, err := db.DB.Query("SELECT kind, COUNT(*) FROM event GROUP BY kind ORDER BY kind")
		if err != nil {
			http.Error(w, fmt.Sprintf("Error querying event kinds: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		stats := make(map[string]int)
		totalCount := 0
		for rows.Next() {
			var kind string
			var count int
			if err := rows.Scan(&kind, &count); err != nil {
				http.Error(w, fmt.Sprintf("Error scanning row: %v", err), http.StatusInternalServerError)
				return
			}
			stats[kind] = count
			totalCount += count
		}

		// Add total count to the stats
		response := map[string]interface{}{
			"total": totalCount,
			"kinds": stats,
		}

		// Encode stats to JSON and write to response
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding JSON: %v", err), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/api/trending/kind20", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// This query joins kind 20 posts with their reactions (kind 7)
		// and counts the number of reactions in the last 24 hours using lateral join
		query := `
			WITH reactions AS (
				SELECT 
					tags_expanded.value->1 #>> '{}' AS original_event_id,
					COUNT(*) as reaction_count
				FROM event e
				CROSS JOIN LATERAL jsonb_array_elements(tags) as tags_expanded(value)
				WHERE e.kind::text = '7' 
				AND e.created_at >= extract(epoch from now() - interval '24 hours')::bigint
				AND tags_expanded.value->0 #>> '{}' = 'e'
				GROUP BY tags_expanded.value->1 #>> '{}'
			)
			SELECT 
				e.id,
				e.pubkey,
				to_timestamp(e.created_at) as created_at,
				e.kind,
				e.content,
				e.tags,
				COALESCE(r.reaction_count, 0) as reaction_count
			FROM event e
			LEFT JOIN reactions r ON e.id = r.original_event_id
			WHERE e.kind::text = '20'
			AND e.created_at >= extract(epoch from now() - interval '24 hours')::bigint
			ORDER BY reaction_count DESC, e.created_at DESC
			LIMIT 20
		`

		rows, err := db.DB.Query(query)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error querying trending posts: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type TrendingPost struct {
			ID            string     `json:"id"`
			PubKey        string     `json:"pubkey"`
			CreatedAt     time.Time  `json:"created_at"`
			Kind          string     `json:"kind"`
			Content       string     `json:"content"`
			Tags          [][]string `json:"tags"`
			ReactionCount int        `json:"reaction_count"`
		}

		var trendingPosts []TrendingPost
		for rows.Next() {
			var post TrendingPost
			var tagsJSON []byte
			if err := rows.Scan(&post.ID, &post.PubKey, &post.CreatedAt, &post.Kind, &post.Content, &tagsJSON, &post.ReactionCount); err != nil {
				http.Error(w, fmt.Sprintf("Error scanning row: %v", err), http.StatusInternalServerError)
				return
			}
			// Parse the tags JSON
			if err := json.Unmarshal(tagsJSON, &post.Tags); err != nil {
				http.Error(w, fmt.Sprintf("Error parsing tags: %v", err), http.StatusInternalServerError)
				return
			}
			trendingPosts = append(trendingPosts, post)
		}

		response := map[string]interface{}{
			"trending": trendingPosts,
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding JSON: %v", err), http.StatusInternalServerError)
		}
	})

	fmt.Println("running on :3334")
	http.ListenAndServe(":3334", relay)
}
