package trending

import (
	"database/sql"
	"encoding/json"
)

// GetTrendingScoreKind20 returns the top 20 trending posts of kind 20 from the last 24 hours
func GetTrendingScoreKind20(db *sql.DB) ([]Post, error) {
	if cached, ok := trendingCache.Get("trending_kind_20"); ok {
		return cached.([]Post), nil
	}

	query := `
		WITH engagement AS (
			SELECT 
				tags_expanded.value->1 #>> '{}' AS original_event_id,
				e.kind::text,
				extract(epoch from now()) - e.created_at as seconds_ago,
				COUNT(*) as count
			FROM event e
			CROSS JOIN LATERAL jsonb_array_elements(tags) as tags_expanded(value)
			WHERE e.kind::text IN ('1', '6', '7')  -- replies, reposts, reactions
			AND e.created_at >= extract(epoch from now() - interval '24 hours')::bigint
			AND tags_expanded.value->0 #>> '{}' = 'e'
			GROUP BY tags_expanded.value->1 #>> '{}', e.kind::text, e.created_at
		),
		scores AS (
			SELECT 
				original_event_id,
				SUM(
					CASE 
						WHEN kind = '7' THEN count * 1.0  -- reactions weight
						WHEN kind = '6' THEN count * 2.0  -- reposts weight
						WHEN kind = '1' THEN count * 1.5  -- replies weight
					END / power((seconds_ago + 7200) / 3600, 1.8)  -- time decay
				) as trending_score
			FROM engagement
			GROUP BY original_event_id
		)
		SELECT 
			e.id,
			e.pubkey,
			to_timestamp(e.created_at) as created_at,
			e.kind,
			e.content,
			e.tags,
			COALESCE(s.trending_score, 0.0) as reaction_count  -- Cast to float
		FROM event e
		LEFT JOIN scores s ON e.id = s.original_event_id
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
