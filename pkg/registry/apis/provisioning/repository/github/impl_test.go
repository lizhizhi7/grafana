package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v70/github"
	mockhub "github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithubClient_GetCommits(t *testing.T) {
	tests := []struct {
		name        string
		mockHandler *http.Client
		owner       string
		repository  string
		branch      string
		path        string
		since       time.Time
		until       time.Time
		wantCommits []Commit
		wantErr     error
	}{
		{
			name: "get commits successfully",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposCommitsByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						commits := []*github.RepositoryCommit{
							{
								SHA: github.Ptr("abc123"),
								Commit: &github.Commit{
									Message: github.Ptr("First commit"),
									Author: &github.CommitAuthor{
										Name:  github.Ptr("Test User"),
										Email: github.Ptr("test@example.com"),
										Date:  &github.Timestamp{Time: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)},
									},
									Committer: &github.CommitAuthor{
										Name:  github.Ptr("Test User"),
										Email: github.Ptr("test@example.com"),
										Date:  &github.Timestamp{Time: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)},
									},
								},
								Author: &github.User{
									Login:     github.Ptr("test-user"),
									AvatarURL: github.Ptr("https://avatar.url"),
								},
								Committer: &github.User{
									Login:     github.Ptr("test-user"),
									AvatarURL: github.Ptr("https://avatar.url"),
								},
							},
							{
								SHA: github.Ptr("def456"),
								Commit: &github.Commit{
									Message: github.Ptr("Second commit"),
									Author: &github.CommitAuthor{
										Name:  github.Ptr("Another User"),
										Email: github.Ptr("another@example.com"),
										Date:  &github.Timestamp{Time: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)},
									},
									Committer: &github.CommitAuthor{
										Name:  github.Ptr("Another User"),
										Email: github.Ptr("another@example.com"),
										Date:  &github.Timestamp{Time: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)},
									},
								},
								Author: &github.User{
									Login:     github.Ptr("another-user"),
									AvatarURL: github.Ptr("https://another.avatar.url"),
								},
								Committer: &github.User{
									Login:     github.Ptr("another-user"),
									AvatarURL: github.Ptr("https://another.avatar.url"),
								},
							},
						}
						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(commits))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			branch:     "main",
			path:       "test.txt",
			since:      time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			until:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC),
			wantCommits: []Commit{
				{
					Ref:     "abc123",
					Message: "First commit",
					Author: &CommitAuthor{
						Name:      "Test User",
						Username:  "test-user",
						AvatarURL: "https://avatar.url",
					},
					Committer: &CommitAuthor{
						Name:      "Test User",
						Username:  "test-user",
						AvatarURL: "https://avatar.url",
					},
					CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				},
				{
					Ref:     "def456",
					Message: "Second commit",
					Author: &CommitAuthor{
						Name:      "Another User",
						Username:  "another-user",
						AvatarURL: "https://another.avatar.url",
					},
					Committer: &CommitAuthor{
						Name:      "Another User",
						Username:  "another-user",
						AvatarURL: "https://another.avatar.url",
					},
					CreatedAt: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC),
				},
			},
			wantErr: nil,
		},
		{
			name: "repository not found",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposCommitsByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusNotFound)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusNotFound,
							},
							Message: "Not Found",
						}))
					}),
				),
			),
			owner:       "test-owner",
			repository:  "nonexistent-repo",
			branch:      "main",
			path:        "test.txt",
			since:       time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			until:       time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC),
			wantCommits: nil,
			wantErr:     ErrResourceNotFound,
		},
		{
			name: "commits missing author",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposCommitsByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						commits := []*github.RepositoryCommit{
							{
								SHA: github.Ptr("abc123"),
								Commit: &github.Commit{
									Message: github.Ptr("First commit"),
									Author: &github.CommitAuthor{
										Name:  github.Ptr("Test User"),
										Date:  &github.Timestamp{Time: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)},
										Email: github.Ptr("test@example.com"),
									},
									Committer: &github.CommitAuthor{
										Name:  github.Ptr("Test User"),
										Date:  &github.Timestamp{Time: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)},
										Email: github.Ptr("test@example.com"),
									},
								},
								// Author is nil
								Committer: &github.User{
									Login:     github.Ptr("test-user"),
									AvatarURL: github.Ptr("https://avatar.url"),
								},
							},
							{
								SHA: github.Ptr("def456"),
								Commit: &github.Commit{
									Message: github.Ptr("Second commit"),
									// Missing Author in Commit
									Committer: &github.CommitAuthor{
										Name:  github.Ptr("Another User"),
										Date:  &github.Timestamp{Time: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)},
										Email: github.Ptr("another@example.com"),
									},
								},
								Author: &github.User{
									Login:     github.Ptr("another-user"),
									AvatarURL: github.Ptr("https://another.avatar.url"),
								},
								Committer: &github.User{
									Login:     github.Ptr("another-user"),
									AvatarURL: github.Ptr("https://another.avatar.url"),
								},
							},
						}
						require.NoError(t, json.NewEncoder(w).Encode(commits))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			branch:     "main",
			path:       "test.txt",
			since:      time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			until:      time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC),
			wantCommits: []Commit{
				{
					Ref:     "abc123",
					Message: "First commit",
					Author: &CommitAuthor{
						Name: "Test User",
						// Username and AvatarURL are empty because Author is nil in the response
					},
					Committer: &CommitAuthor{
						Name:      "Test User",
						Username:  "test-user",
						AvatarURL: "https://avatar.url",
					},
					CreatedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				},
				{
					Ref:     "def456",
					Message: "Second commit",
					Author:  nil, // Author is nil because Commit.Author is nil in the response
					Committer: &CommitAuthor{
						Name:      "Another User",
						Username:  "another-user",
						AvatarURL: "https://another.avatar.url",
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "too many commits",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposCommitsByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						// Return a large number of commits that would exceed the maxCommits limit
						commits := make([]*github.RepositoryCommit, maxCommits+1)
						for i := 0; i < maxCommits+1; i++ {
							commits[i] = &github.RepositoryCommit{
								SHA: github.Ptr(fmt.Sprintf("commit%d", i)),
								Commit: &github.Commit{
									Message: github.Ptr(fmt.Sprintf("Commit %d", i)),
									Author: &github.CommitAuthor{
										Name:  github.Ptr("Test User"),
										Date:  &github.Timestamp{Time: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)},
										Email: github.Ptr("test@example.com"),
									},
								},
								Author: &github.User{
									Login:     github.Ptr("test-user"),
									AvatarURL: github.Ptr("https://avatar.url"),
								},
							}
						}
						require.NoError(t, json.NewEncoder(w).Encode(commits))
					}),
				),
			),
			owner:       "test-owner",
			repository:  "test-repo",
			branch:      "main",
			path:        "test.txt",
			since:       time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			until:       time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC),
			wantCommits: nil,
			wantErr:     fmt.Errorf("too many commits to fetch (more than %d)", maxCommits),
		},
		{
			name: "service unavailable",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposCommitsByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusServiceUnavailable)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusServiceUnavailable,
							},
							Message: "Service unavailable",
						}))
					}),
				),
			),
			owner:       "test-owner",
			repository:  "test-repo",
			branch:      "main",
			path:        "test.txt",
			since:       time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			until:       time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC),
			wantCommits: nil,
			wantErr:     ErrServiceUnavailable,
		},
		{
			name: "other error",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposCommitsByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusInternalServerError,
							},
							Message: "Internal server error",
						}))
					}),
				),
			),
			owner:       "test-owner",
			repository:  "test-repo",
			branch:      "main",
			path:        "test.txt",
			since:       time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			until:       time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC),
			wantCommits: nil,
			wantErr:     errors.New("Internal server error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client
			factory := ProvideFactory()
			factory.Client = tt.mockHandler
			client := factory.New(context.Background(), "")

			// Call the method being tested
			commits, err := client.Commits(context.Background(), tt.owner, tt.repository, tt.branch, tt.path)

			// Check the error
			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(err, tt.wantErr) {
					assert.Equal(t, tt.wantErr, err)
				} else {
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
				assert.Nil(t, commits)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCommits, commits)
			}
		})
	}
}

func TestGithubClient_ListWebhooks(t *testing.T) {
	tests := []struct {
		name         string
		mockHandler  *http.Client
		owner        string
		repository   string
		wantWebhooks []WebhookConfig
		wantErr      error
	}{
		{
			name: "successful webhooks listing",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposHooksByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						hooks := []*github.Hook{
							{
								ID:     github.Ptr(int64(1)),
								Events: []string{"push", "pull_request"},
								Active: github.Ptr(true),
								Config: &github.HookConfig{
									URL:         github.Ptr("https://example.com/webhook1"),
									ContentType: github.Ptr("json"),
								},
							},
							{
								ID:     github.Ptr(int64(2)),
								Events: []string{"issues"},
								Active: github.Ptr(false),
								Config: &github.HookConfig{
									URL:         github.Ptr("https://example.com/webhook2"),
									ContentType: github.Ptr(""),
								},
							},
						}
						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(hooks))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			wantWebhooks: []WebhookConfig{
				{
					ID:          1,
					Events:      []string{"push", "pull_request"},
					Active:      true,
					URL:         "https://example.com/webhook1",
					ContentType: "json",
				},
				{
					ID:          2,
					Events:      []string{"issues"},
					Active:      false,
					URL:         "https://example.com/webhook2",
					ContentType: "form", // Default value when empty
				},
			},
			wantErr: nil,
		},
		{
			name: "empty webhooks list",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposHooksByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						hooks := []*github.Hook{}
						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(hooks))
					}),
				),
			),
			owner:        "test-owner",
			repository:   "test-repo",
			wantWebhooks: []WebhookConfig{},
			wantErr:      nil,
		},
		{
			name: "too many webhooks",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposHooksByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						// Create more webhooks than the maxWebhooks limit
						hooks := make([]*github.Hook, maxWebhooks+1)
						for i := 0; i < maxWebhooks+1; i++ {
							hooks[i] = &github.Hook{
								ID:     github.Ptr(int64(i + 1)),
								Events: []string{"push"},
								Active: github.Ptr(true),
								Config: &github.HookConfig{
									URL:         github.Ptr(fmt.Sprintf("https://example.com/webhook%d", i+1)),
									ContentType: github.Ptr("json"),
								},
							}
						}
						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(hooks))
					}),
				),
			),
			owner:        "test-owner",
			repository:   "test-repo",
			wantWebhooks: nil,
			wantErr:      fmt.Errorf("too many webhooks configured (more than %d)", maxWebhooks),
		},
		{
			name: "service unavailable error",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposHooksByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusServiceUnavailable)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusServiceUnavailable,
							},
							Message: "Service unavailable",
						}))
					}),
				),
			),
			owner:        "test-owner",
			repository:   "test-repo",
			wantWebhooks: nil,
			wantErr:      ErrServiceUnavailable,
		},
		{
			name: "other error",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposHooksByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusInternalServerError,
							},
							Message: "Internal server error",
						}))
					}),
				),
			),
			owner:        "test-owner",
			repository:   "test-repo",
			wantWebhooks: nil,
			wantErr:      errors.New("Internal server error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client
			factory := ProvideFactory()
			factory.Client = tt.mockHandler
			client := factory.New(context.Background(), "")

			// Call the method being tested
			webhooks, err := client.ListWebhooks(context.Background(), tt.owner, tt.repository)

			// Check the error
			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(err, tt.wantErr) {
					assert.Equal(t, tt.wantErr, err)
				} else {
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}

			// Check the result
			assert.Equal(t, tt.wantWebhooks, webhooks)
		})
	}
}

func TestGithubClient_CreateWebhook(t *testing.T) {
	tests := []struct {
		name        string
		mockHandler *http.Client
		owner       string
		repository  string
		config      WebhookConfig
		want        WebhookConfig
		wantErr     error
	}{
		{
			name: "successful webhook creation",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.PostReposHooksByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						// Verify the request body contains the correct webhook config
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)

						hook := &github.Hook{}
						require.NoError(t, json.Unmarshal(body, hook))

						assert.Equal(t, "https://example.com/webhook", hook.Config.GetURL())
						assert.Equal(t, "json", hook.Config.GetContentType())
						assert.Equal(t, "secret123", hook.Config.GetSecret())
						assert.Equal(t, []string{"push", "pull_request"}, hook.Events)
						assert.True(t, hook.GetActive())

						// Return a created hook
						createdHook := &github.Hook{
							ID:     github.Ptr(int64(123)),
							Events: []string{"push", "pull_request"},
							Active: github.Ptr(true),
							Config: &github.HookConfig{
								URL:         github.Ptr("https://example.com/webhook"),
								ContentType: github.Ptr("json"),
								// Secret is not returned by GitHub API
							},
						}

						w.WriteHeader(http.StatusCreated)
						require.NoError(t, json.NewEncoder(w).Encode(createdHook))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			config: WebhookConfig{
				Events:      []string{"push", "pull_request"},
				Active:      true,
				URL:         "https://example.com/webhook",
				ContentType: "json",
				Secret:      "secret123",
			},
			want: WebhookConfig{
				ID:          123,
				Events:      []string{"push", "pull_request"},
				Active:      true,
				URL:         "https://example.com/webhook",
				ContentType: "json",
				Secret:      "secret123",
			},
			wantErr: nil,
		},
		{
			name: "default content type to form",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.PostReposHooksByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)

						hook := &github.Hook{}
						require.NoError(t, json.Unmarshal(body, hook))

						// Verify content type was defaulted to "form"
						assert.Equal(t, "form", hook.Config.GetContentType())

						createdHook := &github.Hook{
							ID:     github.Ptr(int64(123)),
							Events: []string{"push"},
							Active: github.Ptr(true),
							Config: &github.HookConfig{
								URL:         github.Ptr("https://example.com/webhook"),
								ContentType: github.Ptr("form"),
							},
						}

						w.WriteHeader(http.StatusCreated)
						require.NoError(t, json.NewEncoder(w).Encode(createdHook))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			config: WebhookConfig{
				Events: []string{"push"},
				Active: true,
				URL:    "https://example.com/webhook",
				Secret: "secret123",
				// ContentType intentionally omitted
			},
			want: WebhookConfig{
				ID:          123,
				Events:      []string{"push"},
				Active:      true,
				URL:         "https://example.com/webhook",
				ContentType: "form",
				Secret:      "secret123",
			},
			wantErr: nil,
		},
		{
			name: "service unavailable error",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.PostReposHooksByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusServiceUnavailable)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusServiceUnavailable,
							},
							Message: "Service unavailable",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			config: WebhookConfig{
				Events:      []string{"push"},
				Active:      true,
				URL:         "https://example.com/webhook",
				ContentType: "json",
				Secret:      "secret123",
			},
			want:    WebhookConfig{},
			wantErr: ErrServiceUnavailable,
		},
		{
			name: "other error",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.PostReposHooksByOwnerByRepo,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusInternalServerError,
							},
							Message: "Internal server error",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			config: WebhookConfig{
				Events:      []string{"push"},
				Active:      true,
				URL:         "https://example.com/webhook",
				ContentType: "json",
				Secret:      "secret123",
			},
			want:    WebhookConfig{},
			wantErr: errors.New("Internal server error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client
			factory := ProvideFactory()
			factory.Client = tt.mockHandler
			client := factory.New(context.Background(), "")

			// Call the method being tested
			got, err := client.CreateWebhook(context.Background(), tt.owner, tt.repository, tt.config)

			// Check the error
			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(err, tt.wantErr) {
					assert.Equal(t, tt.wantErr, err)
				} else {
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}

			// Check the result
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGithubClient_GetWebhook(t *testing.T) {
	tests := []struct {
		name        string
		mockHandler *http.Client
		owner       string
		repository  string
		webhookID   int64
		want        WebhookConfig
		wantErr     error
	}{
		{
			name: "successful webhook retrieval",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposHooksByOwnerByRepoByHookId,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						hook := &github.Hook{
							ID:     github.Ptr(int64(123)),
							Events: []string{"push", "pull_request"},
							Active: github.Ptr(true),
							Config: &github.HookConfig{
								URL:         github.Ptr("https://example.com/webhook"),
								ContentType: github.Ptr("json"),
								// Secret is not returned by GitHub API
							},
						}
						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(hook))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			webhookID:  123,
			want: WebhookConfig{
				ID:          123,
				Events:      []string{"push", "pull_request"},
				Active:      true,
				URL:         "https://example.com/webhook",
				ContentType: "json",
			},
			wantErr: nil,
		},

		{
			name: "empty content type defaults to json",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposHooksByOwnerByRepoByHookId,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						hook := &github.Hook{
							ID:     github.Ptr(int64(456)),
							Events: []string{"push"},
							Active: github.Ptr(true),
							Config: &github.HookConfig{
								URL:         github.Ptr("https://example.com/webhook-empty-content"),
								ContentType: github.Ptr(""), // Empty content type
							},
						}
						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(hook))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			webhookID:  456,
			want: WebhookConfig{
				ID:          456,
				Events:      []string{"push"},
				Active:      true,
				URL:         "https://example.com/webhook-empty-content",
				ContentType: "json", // Should default to "json"
			},
			wantErr: nil,
		},
		{
			name: "webhook not found",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposHooksByOwnerByRepoByHookId,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusNotFound)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusNotFound,
							},
							Message: "Not Found",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			webhookID:  999,
			want:       WebhookConfig{},
			wantErr:    ErrResourceNotFound,
		},
		{
			name: "service unavailable",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposHooksByOwnerByRepoByHookId,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusServiceUnavailable)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusServiceUnavailable,
							},
							Message: "Service Unavailable",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			webhookID:  123,
			want:       WebhookConfig{},
			wantErr:    ErrServiceUnavailable,
		},
		{
			name: "other error",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposHooksByOwnerByRepoByHookId,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusInternalServerError,
							},
							Message: "Internal server error",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			webhookID:  123,
			want:       WebhookConfig{},
			wantErr:    errors.New("Internal server error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client
			factory := ProvideFactory()
			factory.Client = tt.mockHandler
			client := factory.New(context.Background(), "")

			// Call the method being tested
			got, err := client.GetWebhook(context.Background(), tt.owner, tt.repository, tt.webhookID)

			// Check the error
			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(err, tt.wantErr) {
					assert.Equal(t, tt.wantErr, err)
				} else {
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}

			// Check the result
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGithubClient_DeleteWebhook(t *testing.T) {
	tests := []struct {
		name        string
		mockHandler *http.Client
		owner       string
		repository  string
		webhookID   int64
		wantErr     error
	}{
		{
			name: "successful webhook deletion",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.DeleteReposHooksByOwnerByRepoByHookId,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusNoContent)
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			webhookID:  123,
			wantErr:    nil,
		},
		{
			name: "webhook not found",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.DeleteReposHooksByOwnerByRepoByHookId,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusNotFound)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusNotFound,
							},
							Message: "Not found",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			webhookID:  456,
			wantErr:    ErrResourceNotFound,
		},
		{
			name: "service unavailable",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.DeleteReposHooksByOwnerByRepoByHookId,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusServiceUnavailable)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusServiceUnavailable,
							},
							Message: "Service unavailable",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			webhookID:  789,
			wantErr:    ErrServiceUnavailable,
		},
		{
			name: "other error",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.DeleteReposHooksByOwnerByRepoByHookId,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusInternalServerError,
							},
							Message: "Internal server error",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			webhookID:  101,
			wantErr:    errors.New("Internal server error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client
			factory := ProvideFactory()
			factory.Client = tt.mockHandler
			client := factory.New(context.Background(), "")

			// Call the method being tested
			err := client.DeleteWebhook(context.Background(), tt.owner, tt.repository, tt.webhookID)

			// Check the error
			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(err, tt.wantErr) {
					assert.Equal(t, tt.wantErr, err)
				} else {
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGithubClient_EditWebhook(t *testing.T) {
	tests := []struct {
		name        string
		mockHandler *http.Client
		owner       string
		repository  string
		config      WebhookConfig
		wantErr     error
	}{
		{
			name: "successful webhook edit",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.PatchReposHooksByOwnerByRepoByHookId,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						// Verify the request body contains the correct webhook config
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)

						hook := &github.Hook{}
						require.NoError(t, json.Unmarshal(body, hook))

						assert.Equal(t, "https://example.com/webhook-updated", hook.Config.GetURL())
						assert.Equal(t, "json", hook.Config.GetContentType())
						assert.Equal(t, "updated-secret", hook.Config.GetSecret())
						assert.Equal(t, []string{"push", "pull_request", "issues"}, hook.Events)
						assert.True(t, hook.GetActive())

						// Return the updated hook
						updatedHook := &github.Hook{
							ID:     github.Ptr(int64(123)),
							Events: []string{"push", "pull_request", "issues"},
							Active: github.Ptr(true),
							Config: &github.HookConfig{
								URL:         github.Ptr("https://example.com/webhook-updated"),
								ContentType: github.Ptr("json"),
								// Secret is not returned by GitHub API
							},
						}

						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(updatedHook))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			config: WebhookConfig{
				ID:          123,
				Events:      []string{"push", "pull_request", "issues"},
				Active:      true,
				URL:         "https://example.com/webhook-updated",
				ContentType: "json",
				Secret:      "updated-secret",
			},
			wantErr: nil,
		},
		{
			name: "default content type to form",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.PatchReposHooksByOwnerByRepoByHookId,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						// Verify the request body contains the correct webhook config
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)

						hook := &github.Hook{}
						require.NoError(t, json.Unmarshal(body, hook))

						// Verify content type was defaulted to "form"
						assert.Equal(t, "form", hook.Config.GetContentType())
						assert.Equal(t, "https://example.com/webhook", hook.Config.GetURL())
						assert.Equal(t, "secret123", hook.Config.GetSecret())
						assert.Equal(t, []string{"push"}, hook.Events)
						assert.True(t, hook.GetActive())

						// Return the updated hook
						updatedHook := &github.Hook{
							ID:     github.Ptr(int64(123)),
							Events: []string{"push"},
							Active: github.Ptr(true),
							Config: &github.HookConfig{
								URL:         github.Ptr("https://example.com/webhook"),
								ContentType: github.Ptr("form"),
								// Secret is not returned by GitHub API
							},
						}

						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(updatedHook))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			config: WebhookConfig{
				ID:          123,
				Events:      []string{"push"},
				Active:      true,
				URL:         "https://example.com/webhook",
				ContentType: "", // Empty content type should default to "form"
				Secret:      "secret123",
			},
			wantErr: nil,
		},
		{
			name: "service unavailable error",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.PatchReposHooksByOwnerByRepoByHookId,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusServiceUnavailable)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusServiceUnavailable,
							},
							Message: "Service unavailable",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			config: WebhookConfig{
				ID:          123,
				Events:      []string{"push"},
				Active:      true,
				URL:         "https://example.com/webhook",
				ContentType: "json",
				Secret:      "secret123",
			},
			wantErr: ErrServiceUnavailable,
		},
		{
			name: "other error",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.PatchReposHooksByOwnerByRepoByHookId,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusInternalServerError,
							},
							Message: "Internal server error",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			config: WebhookConfig{
				ID:          123,
				Events:      []string{"push"},
				Active:      true,
				URL:         "https://example.com/webhook",
				ContentType: "json",
				Secret:      "secret123",
			},
			wantErr: errors.New("Internal server error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client
			factory := ProvideFactory()
			factory.Client = tt.mockHandler
			client := factory.New(context.Background(), "")

			// Call the method being tested
			err := client.EditWebhook(context.Background(), tt.owner, tt.repository, tt.config)

			// Check the error
			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(err, tt.wantErr) {
					assert.Equal(t, tt.wantErr, err)
				} else {
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGithubClient_ListPullRequestFiles(t *testing.T) {
	tests := []struct {
		name        string
		mockHandler *http.Client
		owner       string
		repository  string
		number      int
		wantFiles   []CommitFile
		wantErr     error
	}{
		{
			name: "successful pull request files listing",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposPullsFilesByOwnerByRepoByPullNumber,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						files := []*github.CommitFile{
							{
								Filename:  github.Ptr("file1.txt"),
								Additions: github.Ptr(10),
								Deletions: github.Ptr(5),
								Changes:   github.Ptr(15),
								Status:    github.Ptr("modified"),
								Patch:     github.Ptr("@@ -1,5 +1,10 @@"),
							},
							{
								Filename:  github.Ptr("file2.txt"),
								Additions: github.Ptr(20),
								Deletions: github.Ptr(0),
								Changes:   github.Ptr(20),
								Status:    github.Ptr("added"),
								Patch:     github.Ptr("@@ -0,0 +1,20 @@"),
							},
						}
						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(files))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			number:     123,
			wantFiles: []CommitFile{
				&github.CommitFile{
					Filename:  github.Ptr("file1.txt"),
					Additions: github.Ptr(10),
					Deletions: github.Ptr(5),
					Changes:   github.Ptr(15),
					Status:    github.Ptr("modified"),
					Patch:     github.Ptr("@@ -1,5 +1,10 @@"),
				},
				&github.CommitFile{
					Filename:  github.Ptr("file2.txt"),
					Additions: github.Ptr(20),
					Deletions: github.Ptr(0),
					Changes:   github.Ptr(20),
					Status:    github.Ptr("added"),
					Patch:     github.Ptr("@@ -0,0 +1,20 @@"),
				},
			},
			wantErr: nil,
		},
		{
			name: "empty files list",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposPullsFilesByOwnerByRepoByPullNumber,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						files := []*github.CommitFile{}
						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(files))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			number:     456,
			wantFiles:  []CommitFile{},
			wantErr:    nil,
		},
		{
			name: "too many files",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposPullsFilesByOwnerByRepoByPullNumber,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						// Create more files than the maxPRFiles limit
						files := make([]*github.CommitFile, maxPRFiles+1)
						for i := 0; i < maxPRFiles+1; i++ {
							files[i] = &github.CommitFile{
								Filename:  github.Ptr(fmt.Sprintf("file%d.txt", i+1)),
								Additions: github.Ptr(i + 1),
								Deletions: github.Ptr(0),
								Changes:   github.Ptr(i + 1),
								Status:    github.Ptr("added"),
							}
						}
						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(files))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			number:     789,
			wantFiles:  nil,
			wantErr:    fmt.Errorf("pull request contains too many files (more than %d)", maxPRFiles),
		},
		{
			name: "service unavailable error",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposPullsFilesByOwnerByRepoByPullNumber,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusServiceUnavailable)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusServiceUnavailable,
							},
							Message: "Service unavailable",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			number:     101,
			wantFiles:  nil,
			wantErr:    ErrServiceUnavailable,
		},
		{
			name: "other error",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.GetReposPullsFilesByOwnerByRepoByPullNumber,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusInternalServerError,
							},
							Message: "Internal server error",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			number:     202,
			wantFiles:  nil,
			wantErr:    errors.New("Internal server error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client
			factory := ProvideFactory()
			factory.Client = tt.mockHandler
			client := factory.New(context.Background(), "")

			// Call the method being tested
			files, err := client.ListPullRequestFiles(context.Background(), tt.owner, tt.repository, tt.number)

			// Check the error
			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(err, tt.wantErr) {
					assert.Equal(t, tt.wantErr, err)
				} else {
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}

			// Check the result
			assert.Equal(t, tt.wantFiles, files)
		})
	}
}

func TestCreatePullRequestComment(t *testing.T) {
	tests := []struct {
		name        string
		mockHandler *http.Client
		owner       string
		repository  string
		number      int
		body        string
		wantErr     error
	}{
		{
			name: "successful comment creation",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.PostReposIssuesCommentsByOwnerByRepoByIssueNumber,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						// Verify the request body contains the correct comment
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)

						comment := &github.IssueComment{}
						require.NoError(t, json.Unmarshal(body, comment))
						assert.Equal(t, "Test comment", comment.GetBody())

						// Return the created comment
						createdComment := &github.IssueComment{
							ID:   github.Ptr(int64(123)),
							Body: github.Ptr("Test comment"),
						}

						w.WriteHeader(http.StatusCreated)
						require.NoError(t, json.NewEncoder(w).Encode(createdComment))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			number:     101,
			body:       "Test comment",
			wantErr:    nil,
		},
		{
			name: "service unavailable error",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.PostReposIssuesCommentsByOwnerByRepoByIssueNumber,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusServiceUnavailable)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusServiceUnavailable,
							},
							Message: "Service unavailable",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			number:     101,
			body:       "Test comment",
			wantErr:    ErrServiceUnavailable,
		},
		{
			name: "other error",
			mockHandler: mockhub.NewMockedHTTPClient(
				mockhub.WithRequestMatchHandler(
					mockhub.PostReposIssuesCommentsByOwnerByRepoByIssueNumber,
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusInternalServerError)
						require.NoError(t, json.NewEncoder(w).Encode(github.ErrorResponse{
							Response: &http.Response{
								StatusCode: http.StatusInternalServerError,
							},
							Message: "Internal server error",
						}))
					}),
				),
			),
			owner:      "test-owner",
			repository: "test-repo",
			number:     101,
			body:       "Test comment",
			wantErr:    errors.New("Internal server error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock client
			factory := ProvideFactory()
			factory.Client = tt.mockHandler
			client := factory.New(context.Background(), "")

			// Call the method being tested
			err := client.CreatePullRequestComment(context.Background(), tt.owner, tt.repository, tt.number, tt.body)

			// Check the error
			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(err, tt.wantErr) {
					assert.Equal(t, tt.wantErr, err)
				} else {
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPaginatedList(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func() (func(context.Context, *github.ListOptions) ([]string, *github.Response, error), listOptions)
		want      []string
		wantErr   error
	}{
		{
			name: "single page",
			mockSetup: func() (func(context.Context, *github.ListOptions) ([]string, *github.Response, error), listOptions) {
				items := []string{"item1", "item2", "item3"}
				listFn := func(_ context.Context, _ *github.ListOptions) ([]string, *github.Response, error) {
					return items, &github.Response{
						NextPage: 0,
					}, nil
				}
				return listFn, defaultListOptions(100)
			},
			want:    []string{"item1", "item2", "item3"},
			wantErr: nil,
		},
		{
			name: "multiple pages",
			mockSetup: func() (func(context.Context, *github.ListOptions) ([]string, *github.Response, error), listOptions) {
				page1 := []string{"item1", "item2"}
				page2 := []string{"item3", "item4"}
				page3 := []string{"item5"}

				var callCount int
				listFn := func(_ context.Context, opts *github.ListOptions) ([]string, *github.Response, error) {
					callCount++
					switch callCount {
					case 1:
						return page1, &github.Response{
							NextPage: 2,
						}, nil
					case 2:
						assert.Equal(t, 2, opts.Page)
						return page2, &github.Response{
							NextPage: 3,
						}, nil
					case 3:
						assert.Equal(t, 3, opts.Page)
						return page3, &github.Response{
							NextPage: 0,
						}, nil
					default:
						return nil, nil, errors.New("unexpected call")
					}
				}
				return listFn, defaultListOptions(100)
			},
			want:    []string{"item1", "item2", "item3", "item4", "item5"},
			wantErr: nil,
		},
		{
			name: "error on first page",
			mockSetup: func() (func(context.Context, *github.ListOptions) ([]string, *github.Response, error), listOptions) {
				listFn := func(_ context.Context, _ *github.ListOptions) ([]string, *github.Response, error) {
					return nil, &github.Response{}, errors.New("API error")
				}
				return listFn, defaultListOptions(100)
			},
			want:    nil,
			wantErr: errors.New("API error"),
		},
		{
			name: "service unavailable error",
			mockSetup: func() (func(context.Context, *github.ListOptions) ([]string, *github.Response, error), listOptions) {
				listFn := func(_ context.Context, _ *github.ListOptions) ([]string, *github.Response, error) {
					return nil, &github.Response{}, &github.ErrorResponse{
						Response: &http.Response{
							StatusCode: http.StatusServiceUnavailable,
						},
					}
				}
				return listFn, defaultListOptions(100)
			},
			want:    nil,
			wantErr: ErrServiceUnavailable,
		},
		{
			name: "resource not found error",
			mockSetup: func() (func(context.Context, *github.ListOptions) ([]string, *github.Response, error), listOptions) {
				listFn := func(_ context.Context, _ *github.ListOptions) ([]string, *github.Response, error) {
					return nil, &github.Response{}, &github.ErrorResponse{
						Response: &http.Response{
							StatusCode: http.StatusNotFound,
						},
					}
				}
				return listFn, defaultListOptions(100)
			},
			want:    nil,
			wantErr: ErrResourceNotFound,
		},
		{
			name: "too many items error",
			mockSetup: func() (func(context.Context, *github.ListOptions) ([]string, *github.Response, error), listOptions) {
				listFn := func(_ context.Context, _ *github.ListOptions) ([]string, *github.Response, error) {
					// Return more items than the max allowed
					items := make([]string, 10)
					for i := range items {
						items[i] = fmt.Sprintf("item%d", i+1)
					}
					return items, &github.Response{
						NextPage: 2,
					}, nil
				}
				return listFn, listOptions{
					ListOptions: github.ListOptions{
						Page:    1,
						PerPage: 100,
					},
					MaxItems: 5, // Set max items to less than what we'll return
				}
			},
			want:    nil,
			wantErr: ErrTooManyItems,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listFn, opts := tt.mockSetup()

			got, err := paginatedList(context.Background(), listFn, opts)

			if tt.wantErr != nil {
				assert.Error(t, err)
				if errors.Is(err, tt.wantErr) {
					assert.Equal(t, tt.wantErr, err)
				} else {
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestDefaultListOptions(t *testing.T) {
	tests := []struct {
		name     string
		maxItems int
		want     listOptions
	}{
		{
			name:     "with zero max items",
			maxItems: 0,
			want: listOptions{
				ListOptions: github.ListOptions{
					Page:    1,
					PerPage: 100,
				},
				MaxItems: 0,
			},
		},
		{
			name:     "with positive max items",
			maxItems: 50,
			want: listOptions{
				ListOptions: github.ListOptions{
					Page:    1,
					PerPage: 100,
				},
				MaxItems: 50,
			},
		},
		{
			name:     "with large max items",
			maxItems: 1000,
			want: listOptions{
				ListOptions: github.ListOptions{
					Page:    1,
					PerPage: 100,
				},
				MaxItems: 1000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultListOptions(tt.maxItems)
			assert.Equal(t, tt.want, got)
		})
	}
}
