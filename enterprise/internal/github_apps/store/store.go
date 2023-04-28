package store

import (
	"context"

	"github.com/keegancsmith/sqlf"

	"github.com/sourcegraph/sourcegraph/enterprise/internal/github_apps/types"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
	encryption "github.com/sourcegraph/sourcegraph/internal/encryption"
	"github.com/sourcegraph/sourcegraph/internal/encryption/keyring"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

// GitHubAppsStore handles storing and retrieving GitHub Apps from the database.
type GitHubAppsStore interface {
	// Create inserts a new GitHub App into the database.
	Create(ctx context.Context, app *types.GitHubApp) (int, error)

	// Delete removes a GitHub App from the database by ID.
	Delete(ctx context.Context, id int) error

	// Update updates a GitHub App in the database and returns the updated struct.
	Update(ctx context.Context, id int, app *types.GitHubApp) (*types.GitHubApp, error)

	// GetByID retrieves a GitHub App from the database by ID.
	GetByID(ctx context.Context, id int) (*types.GitHubApp, error)

	// GetByAppID retrieves a GitHub App from the database by appID and base url
	GetByAppID(ctx context.Context, appID int, baseURL string) (*types.GitHubApp, error)

	// GetBySlug retrieves a GitHub App from the database by slug and base url
	GetBySlug(ctx context.Context, slug string, baseURL string) (*types.GitHubApp, error)

	// WithEncryptionKey sets encryption key on store. Returns a new GitHubAppsStore
	WithEncryptionKey(key encryption.Key) GitHubAppsStore
}

// gitHubAppStore handles storing and retrieving GitHub Apps from the database.
type gitHubAppsStore struct {
	*basestore.Store

	key encryption.Key
}

func GitHubAppsWith(other *basestore.Store) GitHubAppsStore {
	return &gitHubAppsStore{
		Store: basestore.NewWithHandle(other.Handle()),
	}
}

// WithEncryptionKey sets encryption key on store. Returns a new GitHubAppsStore
func (s *gitHubAppsStore) WithEncryptionKey(key encryption.Key) GitHubAppsStore {
	return &gitHubAppsStore{Store: s.Store, key: key}
}

func (s *gitHubAppsStore) getEncryptionKey() encryption.Key {
	if s.key != nil {
		return s.key
	}
	return keyring.Default().GitHubAppKey
}

// Create inserts a new GitHub App into the database.
func (s *gitHubAppsStore) Create(ctx context.Context, app *types.GitHubApp) (int, error) {
	key := s.getEncryptionKey()
	clientSecret, _, err := encryption.MaybeEncrypt(ctx, key, app.ClientSecret)
	if err != nil {
		return -1, err
	}
	privateKey, keyID, err := encryption.MaybeEncrypt(ctx, key, app.PrivateKey)
	if err != nil {
		return -1, err
	}

	query := sqlf.Sprintf(`INSERT INTO
	    github_apps (app_id, name, slug, base_url, client_id, client_secret, private_key, encryption_key_id, logo)
    	VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
		RETURNING id`,
		app.AppID, app.Name, app.Slug, app.BaseURL, app.ClientID, clientSecret, privateKey, keyID, app.Logo)
	id, _, err := basestore.ScanFirstInt(s.Query(ctx, query))
	return id, err
}

// Delete removes a GitHub App from the database by ID.
func (s *gitHubAppsStore) Delete(ctx context.Context, id int) error {
	query := sqlf.Sprintf(`DELETE FROM github_apps WHERE id = %s`, id)
	return s.Exec(ctx, query)
}

var scanGitHubApp = basestore.NewFirstScanner(func(s dbutil.Scanner) (*types.GitHubApp, error) {
	var app types.GitHubApp

	err := s.Scan(
		&app.ID,
		&app.AppID,
		&app.Name,
		&app.Slug,
		&app.BaseURL,
		&app.ClientID,
		&app.ClientSecret,
		&app.PrivateKey,
		&app.EncryptionKey,
		&app.Logo,
		&app.CreatedAt,
		&app.UpdatedAt)
	return &app, err
})

func (s *gitHubAppsStore) decrypt(ctx context.Context, app *types.GitHubApp) (*types.GitHubApp, error) {
	key := s.getEncryptionKey()
	var cs, pk string

	cs, err := encryption.MaybeDecrypt(ctx, key, app.ClientSecret, app.EncryptionKey)
	if err != nil {
		return nil, err
	}
	app.ClientSecret = cs
	pk, err = encryption.MaybeDecrypt(ctx, key, app.PrivateKey, app.EncryptionKey)
	if err != nil {
		return nil, err
	}
	app.PrivateKey = pk

	return app, nil
}

// Update updates a GitHub App in the database and returns the updated struct.
func (s *gitHubAppsStore) Update(ctx context.Context, id int, app *types.GitHubApp) (*types.GitHubApp, error) {
	key := s.getEncryptionKey()
	clientSecret, _, err := encryption.MaybeEncrypt(ctx, key, app.ClientSecret)
	if err != nil {
		return nil, err
	}
	privateKey, keyID, err := encryption.MaybeEncrypt(ctx, key, app.PrivateKey)
	if err != nil {
		return nil, err
	}

	query := sqlf.Sprintf(`UPDATE github_apps
             SET app_id = %s, name = %s, slug = %s, base_url = %s, client_id = %s, client_secret = %s, private_key = %s, encryption_key_id = %s, logo = %s, updated_at = NOW()
             WHERE id = %s
			 RETURNING id, app_id, name, slug, base_url, client_id, client_secret, private_key, encryption_key_id, logo, created_at, updated_at`,
		app.AppID, app.Name, app.Slug, app.BaseURL, app.ClientID, clientSecret, privateKey, keyID, app.Logo, id)
	app, ok, err := scanGitHubApp(s.Query(ctx, query))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.Newf("cannot update app with id: %d because no such app exists", id)
	}
	return s.decrypt(ctx, app)
}

func (s *gitHubAppsStore) get(ctx context.Context, where *sqlf.Query) (*types.GitHubApp, error) {
	selectQuery := `SELECT
		id,
		app_id,
		name,
		slug,
		base_url,
		client_id,
		client_secret,
		private_key,
		encryption_key_id,
		logo,
		created_at,
		updated_at
	FROM github_apps
	WHERE %s`

	query := sqlf.Sprintf(selectQuery, where)
	app, ok, err := scanGitHubApp(s.Query(ctx, query))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.Newf("no app exists matching criteria: %v", *where)
	}

	return s.decrypt(ctx, app)
}

// GetByID retrieves a GitHub App from the database by ID.
func (s *gitHubAppsStore) GetByID(ctx context.Context, id int) (*types.GitHubApp, error) {
	return s.get(ctx, sqlf.Sprintf(`id = %s`, id))
}

// GetByAppID retrieves a GitHub App from the database by appID and base url
func (s *gitHubAppsStore) GetByAppID(ctx context.Context, appID int, baseURL string) (*types.GitHubApp, error) {
	return s.get(ctx, sqlf.Sprintf(`app_id = %s AND base_url = %s`, appID, baseURL))
}

// GetBySlug retrieves a GitHub App from the database by slug and base url
func (s *gitHubAppsStore) GetBySlug(ctx context.Context, slug string, baseURL string) (*types.GitHubApp, error) {
	return s.get(ctx, sqlf.Sprintf(`slug = %s AND base_url = %s`, slug, baseURL))
}
