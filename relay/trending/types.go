package trending

import "time"

type Post struct {
	ID            string     `json:"id"`
	PubKey        string     `json:"pubkey"`
	CreatedAt     time.Time  `json:"created_at"`
	Kind          int        `json:"kind"`
	Content       string     `json:"content"`
	Tags          [][]string `json:"tags"`
	ReactionCount float64    `json:"reaction_count"`
}
