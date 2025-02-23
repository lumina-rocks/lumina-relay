package trending

import (
	"database/sql"
	"encoding/json"
	"time"

	"git.highperfocused.tech/highperfocused/lumina-relay/relay/cache"
)

var (
	trendingCache = cache.New()
	cacheDuration = 5 * time.Minute
)

// GetTrendingBasicKind20 returns the top 20 trending posts of kind 20 from the last 24 hours
func GetTrendingBasicKind20(db *sql.DB) ([]Post, error) {
	if cached, ok := trendingCache.Get("trending_kind_20"); ok {
		return cached.([]Post), nil
	}

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

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trendingPosts []Post
	for rows.Next() {
		var post Post
		var tagsJSON []byte
		if err := rows.Scan(&post.ID, &post.PubKey, &post.CreatedAt, &post.Kind, &post.Content, &tagsJSON, &post.ReactionCount); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(tagsJSON, &post.Tags); err != nil {
			return nil, err
		}
		trendingPosts = append(trendingPosts, post)
	}

	trendingCache.Set("trending_kind_20", trendingPosts, cacheDuration)
	return trendingPosts, nil
}
