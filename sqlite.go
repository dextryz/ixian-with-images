package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"
	"fmt"

	"github.com/dextryz/nostr"

	_ "github.com/mattn/go-sqlite3"
)

var ErrDupEvent = errors.New("duplicate: event already exists")
var ErrDupProfile = errors.New("duplicate: profile already exists")

type Profile struct {
	PubKey     string
	Name       string
	About      string
	Website    string
	Banner     string
	Picture    string
	Identifier string
}

// We want the client to have its own domain language to make
// it explicit what is supported and what is not. We are flattening a general NIP-23 event.
// Store both content for reference. Also makes it more explicit. Principle of Explicivity
type Article struct {
	Id          string
	Image       string
	Title       string
	Summary     string
	HashTags    []string // #focus #think without to the # in sstring
	MdContent   string
	HtmlContent string
	PublishedAt string
}

type Db struct {
	*sql.DB
    QueryIdLimit int
    QueryAuthorLimit int
    QueryTagLimit int
}

func (s *Db) Close() {
	s.DB.Close()
}

func createTables(db *sql.DB) error {

	createArticleSQL := `
    CREATE TABLE IF NOT EXISTS article (
        article_id TEXT PRIMARY KEY,
        image TEXT,
        title TEXT,
        summary TEXT,
        md_content TEXT,
        html_content TEXT,
        published_at INTEGER
    );`

	createTagSQL := `
    CREATE TABLE IF NOT EXISTS hashtag (
        hashtag_name TEXT PRIMARY KEY
    );`

	createArticleHashtagSQL := `
    CREATE TABLE IF NOT EXISTS article_hashtag (
        article_id TEXT,
        hashtag_name TEXT,
        FOREIGN KEY (article_id) REFERENCES article (article_id),
        FOREIGN KEY (hashtag_name) REFERENCES hashtag (hashtag_name),
        PRIMARY KEY (article_id, hashtag_name)
    );`

	createProfileSQL := `
    CREATE TABLE IF NOT EXISTS profile (
        pubkey TEXT PRIMARY KEY,
        name TEXT,
        about TEXT,
        website TEXT,
        banner TEXT,
        picture TEXT,
        identifier TEXT
    );`

	createArticleProfileSQL := `
    CREATE TABLE IF NOT EXISTS article_profile (
        article_id TEXT,
        pubkey TEXT,
        FOREIGN KEY (article_id) REFERENCES article (article_id),
        FOREIGN KEY (pubkey) REFERENCES profile (pubkey),
        PRIMARY KEY (article_id, pubkey)
    )
    `

	_, err := db.Exec(createProfileSQL)
	if err != nil {
		return err
	}

	_, err = db.Exec(createArticleProfileSQL)
	if err != nil {
		return err
	}

	_, err = db.Exec(createArticleSQL)
	if err != nil {
		return err
	}

	_, err = db.Exec(createTagSQL)
	if err != nil {
		return err
	}

	_, err = db.Exec(createArticleHashtagSQL)
	if err != nil {
		return err
	}

	log.Println("table events created")

	return nil
}

func NewSqlite(database string) *Db {

	db, err := sql.Open("sqlite3", database)
	if err != nil {
		log.Fatal(err)
	}

	err = createTables(db)
	if err != nil {
		log.Fatal(err)
	}

	return &Db{
		DB:         db,
		QueryIdLimit: 500,
		QueryAuthorLimit: 10,
		QueryTagLimit: 10,
	}
}

func (s *Db) StoreProfile(ctx context.Context, p *nostr.Profile, pubkey string) (*Profile, error) {

	profile := &Profile{
		PubKey:     pubkey,
		Name:       p.Name,
		About:      p.About,
		Website:    p.Website,
		Banner:     p.Banner,
		Picture:    p.Picture,
		Identifier: p.Nip05,
	}

	err := s.insertProfile(ctx, profile)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

// THis funtion is responsible for data convertion.
// Has to convert data from nostr DL to db DL.
func (s *Db) StoreArticle(ctx context.Context, e *nostr.Event) (*Article, error) {

	// Sample Unix timestamp: 1635619200 (represents 2021-10-30)
	unixTimestamp := int64(e.CreatedAt)

	// Convert Unix timestamp to time.Time
	t := time.Unix(unixTimestamp, 0)

	// Format time.Time to "yyyy-mm-dd"
	createdAt := t.Format("2006-01-02")

	// 	// Encode NIP-01 event id to NIP-19 note id
	// 	id, err := nostr.EncodeNote(e.Id)
	// 	if err != nil {
	// 		return err
	// 	}

	a := &Article{
		Id:          e.Id,
		MdContent:   e.Content,
		HtmlContent: e.Content,
		PublishedAt: createdAt,
	}

	for _, t := range e.Tags {
		if t.Key() == "image" {
			a.Image = t.Value()
		}
		if t.Key() == "title" {
			a.Title = t.Value()
		}
		if t.Key() == "summary" {
			a.Summary = t.Value()
		}
		// TODO: Check the # prefix and filter in tags.
		if t.Key() == "t" {
			a.HashTags = append(a.HashTags, t.Value())
		}
	}

	err := s.insertArticle(ctx, a)
	if err != nil {
		return nil, err
	}

	err = s.associateProfile(ctx, a.Id, e.PubKey)
	if err != nil {
		return nil, err
	}

	for _, tag := range a.HashTags {
		err = s.insertAndAssociateTag(ctx, a.Id, tag)
		if err != nil {
			return nil, err
		}
	}

	log.Printf("Event (id: %s) stored in repository DB", e.Id)

	return a, nil
}

func (s *Db) insertProfile(ctx context.Context, p *Profile) error {

	eventSql := "INSERT INTO profile (pubkey, name, about, website, banner, picture, identifier) VALUES ($1, $2, $3, $4, $5, $6, $7)"

	res, err := s.DB.ExecContext(ctx, eventSql, p.PubKey, p.Name, p.About, p.Website, p.Banner, p.Picture, p.Identifier)
	if err != nil {
		return err
	}

	nr, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if nr == 0 {
		return ErrDupProfile
	}

	return nil
}

func (s *Db) insertArticle(ctx context.Context, a *Article) error {

	eventSql := "INSERT INTO article (article_id, image, title, summary, md_content, html_content, published_at) VALUES ($1, $2, $3, $4, $5, $6, $7)"

	res, err := s.DB.ExecContext(ctx, eventSql, a.Id, a.Image, a.Title, a.Summary, a.MdContent, a.HtmlContent, a.PublishedAt)
	if err != nil {
		return err
	}

	nr, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if nr == 0 {
		return ErrDupEvent
	}

	return nil
}

func (s *Db) associateProfile(ctx context.Context, noteId string, pubkey string) error {

	// Associate profile with article
	_, err := s.DB.ExecContext(ctx, "INSERT INTO article_profile (article_id, pubkey) VALUES (?, ?)", noteId, pubkey)
	if err != nil {
		return err
	}

	return nil
}

// TODO: Why do I need to pass the context?
func (s *Db) insertAndAssociateTag(ctx context.Context, noteId string, tagName string) error {

	// Insert tag (ignore if already exists)
	_, err := s.DB.ExecContext(ctx, "INSERT OR IGNORE INTO hashtag (hashtag_name) VALUES (?)", tagName)
	if err != nil {
		return err
	}

	// Associate tag with note
	_, err = s.DB.ExecContext(ctx, "INSERT INTO article_hashtag (article_id, hashtag_name) VALUES (?, ?)", noteId, tagName)
	if err != nil {
		return err
	}

	return nil
}

func (s *Db) QueryArticles(ctx context.Context, filter nostr.Filter) ([]*Article, error) {

    articles := []*Article{}

    // 1. Search by IDs

    if filter.Ids != nil {
        if len(filter.Ids) > s.QueryIdLimit {
            return nil, fmt.Errorf("requested articles exceeds ID limit of %d", s.QueryIdLimit)
        }
    }

    // 2. Search by PubKeys

    if filter.Authors != nil {
        if len(filter.Ids) > s.QueryAuthorLimit {
            return nil, fmt.Errorf("authors exceeds limit of %d", s.QueryAuthorLimit)
        }

    }

    // 3. Search by Tags

    for _, tags := range filter.Tags {
        if len(tags) > s.QueryTagLimit {
            return nil, fmt.Errorf("tags exceeds limit of %d", s.QueryTagLimit)
        }
    }

    // 4. Search by Content

    return articles, nil
}

func (s *Db) queryProfileByPubkey(pubkey string) (*Profile, error) {

	rows := s.DB.QueryRow(`SELECT * FROM profile WHERE pubkey = ?`, pubkey)

	var p Profile
	err := rows.Scan(&p.PubKey, &p.Name, &p.About, &p.Website, &p.Banner, &p.Picture, &p.Identifier)
	if err != nil {
		return nil, err
	}

	log.Println("Found profile by pubkey:", p)

	return &p, nil
}

func (s *Db) queryArticleById(tag string) (*Article, error) {

	rows := s.DB.QueryRow(`SELECT * FROM article WHERE article_id = ?`, tag)

	var a Article
	err := rows.Scan(&a.Id, &a.Image, &a.Title, &a.Summary, &a.MdContent, &a.HtmlContent, &a.PublishedAt)
	if err != nil {
		return nil, err
	}
	log.Println("Found Note by ID:", a)

	return &a, nil
}

func (s *Db) queryArticleByTag(tag string) ([]*Article, error) {

	rows, err := s.DB.Query(`
        SELECT n.* FROM article n
        JOIN article_hashtag nt ON n.article_id = nt.article_id
        JOIN hashtag t ON nt.hashtag_name = t.hashtag_name
        WHERE t.hashtag_name = ?
    `, tag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

    articles := []*Article{}
	for rows.Next() {
		var a Article
		err := rows.Scan(&a.Id, &a.Image, &a.Title, &a.Summary, &a.MdContent, &a.HtmlContent, &a.PublishedAt)
		if err != nil {
			return nil, err
		}
        articles = append(articles, &a)
	}

	return articles, nil
}

func (s *Db) queryArticleByProfile(pubkey string) error {

	rows, err := s.DB.Query(`
        SELECT n.* FROM article n
        JOIN article_profile nt ON n.article_id = nt.article_id
        JOIN profile t ON nt.pubkey = t.pubkey
        WHERE t.pubkey = ?
    `, pubkey)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var a Article
		err := rows.Scan(&a.Id, &a.Image, &a.Title, &a.Summary, &a.MdContent, &a.HtmlContent, &a.PublishedAt)
		if err != nil {
			return err
		}
		log.Println("Found article by profile:", a)
	}

	return nil
}

func (s *Db) queryProfileByArticle(id string) (*Profile, error) {

	rows := s.DB.QueryRow(`
        SELECT n.* FROM profile n
        JOIN article_profile nt ON n.pubkey = nt.pubkey
        JOIN article t ON nt.article_id = t.article_id
        WHERE t.article_id = ?
    `, id)

    var p Profile

    err := rows.Scan(&p.PubKey, &p.Name, &p.About, &p.Website, &p.Banner, &p.Picture, &p.Identifier)
    if err != nil {
        return nil, err
    }

	return &p, nil
}
