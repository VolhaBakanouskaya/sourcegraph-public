package graphql

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sourcegraph/log/logtest"

	store "github.com/sourcegraph/sourcegraph/internal/codeintel/stores/dbstore"
	shared "github.com/sourcegraph/sourcegraph/internal/codeintel/symbols/shared"
	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/observation"
	"github.com/sourcegraph/sourcegraph/internal/types"
)

var (
	testRange1 = shared.Range{Start: shared.Position{Line: 11, Character: 21}, End: shared.Position{Line: 31, Character: 41}}
	testRange2 = shared.Range{Start: shared.Position{Line: 12, Character: 22}, End: shared.Position{Line: 32, Character: 42}}
	testRange3 = shared.Range{Start: shared.Position{Line: 13, Character: 23}, End: shared.Position{Line: 33, Character: 43}}
	testRange4 = shared.Range{Start: shared.Position{Line: 14, Character: 24}, End: shared.Position{Line: 34, Character: 44}}
	testRange5 = shared.Range{Start: shared.Position{Line: 15, Character: 25}, End: shared.Position{Line: 35, Character: 45}}
)

func TestDefinitions(t *testing.T) {
	// Set up
	logger := logtest.Scoped(t)
	sqlDB := dbtest.NewDB(logger, t)
	db := database.NewDB(logger, sqlDB)
	mockGitServer := gitserver.NewClient(db)
	mockGitserverClient := NewMockGitserverClient()
	mockRepo := &types.Repo{}
	mockSvc := NewMockService()

	resolver := New(mockSvc, 50, &observation.TestContext)
	resolver.SetLocalCommitCache(mockGitserverClient)
	resolver.SetLocalGitTreeTranslator(mockGitServer, mockRepo, "deadbeef", "s1/main.go")

	locations := []shared.Location{
		{DumpID: 51, Path: "a.go", Range: testRange1},
		{DumpID: 51, Path: "b.go", Range: testRange2},
		{DumpID: 51, Path: "a.go", Range: testRange3},
		{DumpID: 51, Path: "b.go", Range: testRange4},
		{DumpID: 51, Path: "c.go", Range: testRange5},
	}
	mockSvc.GetDefinitionsFunc.PushReturn(locations, len(locations), nil)

	uploads := []store.Dump{
		{ID: 50, Commit: "deadbeef", Root: "sub1/"},
		{ID: 51, Commit: "deadbeef", Root: "sub2/"},
		{ID: 52, Commit: "deadbeef", Root: "sub3/"},
		{ID: 53, Commit: "deadbeef", Root: "sub4/"},
	}
	resolver.SetUploadsDataLoader(uploads)

	mockRequest := shared.RequestArgs{
		RepositoryID: 51,
		Commit:       "deadbeef",
		Path:         "s1/main.go",
		Line:         10,
		Character:    20,
	}
	adjustedLocations, err := resolver.Definitions(context.Background(), mockRequest)
	if err != nil {
		t.Fatalf("unexpected error querying definitions: %s", err)
	}
	sharedUploads := storeDumpToSymbolDump(uploads)
	expectedLocations := []shared.UploadLocation{
		{Dump: sharedUploads[1], Path: "sub2/a.go", TargetCommit: "deadbeef", TargetRange: testRange1},
		{Dump: sharedUploads[1], Path: "sub2/b.go", TargetCommit: "deadbeef", TargetRange: testRange2},
		{Dump: sharedUploads[1], Path: "sub2/a.go", TargetCommit: "deadbeef", TargetRange: testRange3},
		{Dump: sharedUploads[1], Path: "sub2/b.go", TargetCommit: "deadbeef", TargetRange: testRange4},
		{Dump: sharedUploads[1], Path: "sub2/c.go", TargetCommit: "deadbeef", TargetRange: testRange5},
	}
	if diff := cmp.Diff(expectedLocations, adjustedLocations); diff != "" {
		t.Errorf("unexpected locations (-want +got):\n%s", diff)
	}
}

// func TestDefinitionsWithSubRepoPermissions(t *testing.T) {
// 	t.Skip("Different arch for now, need to come back to this.")
// 	mockDBStore := NewMockDBStore()
// 	mockLSIFStore := NewMockLSIFStore()
// 	mockGitserverClient := NewMockGitserverClient()
// 	mockPositionAdjuster := noopPositionAdjuster()
// 	mockSymbolsResolver := NewMockSymbolsResolver()

// 	locations := []lsifstore.Location{
// 		{DumpID: 51, Path: "a.go", Range: testRange1},
// 		{DumpID: 51, Path: "b.go", Range: testRange2},
// 		{DumpID: 51, Path: "a.go", Range: testRange3},
// 		{DumpID: 51, Path: "b.go", Range: testRange4},
// 		{DumpID: 51, Path: "c.go", Range: testRange5},
// 	}
// 	mockLSIFStore.DefinitionsFunc.PushReturn(nil, 0, nil)
// 	mockLSIFStore.DefinitionsFunc.PushReturn(locations, len(locations), nil)

// 	uploads := []dbstore.Dump{
// 		{ID: 50, Commit: "deadbeef", Root: "sub1/"},
// 		{ID: 51, Commit: "deadbeef", Root: "sub2/"},
// 		{ID: 52, Commit: "deadbeef", Root: "sub3/"},
// 		{ID: 53, Commit: "deadbeef", Root: "sub4/"},
// 	}

// 	// Applying sub-repo permissions
// 	// TODO: I don't have the same architecture, need to come back to this.
// 	checker := authz.NewMockSubRepoPermissionChecker()

// 	checker.EnabledFunc.SetDefaultHook(func() bool {
// 		return true
// 	})

// 	checker.PermissionsFunc.SetDefaultHook(func(ctx context.Context, i int32, content authz.RepoContent) (authz.Perms, error) {
// 		if content.Path == "sub2/a.go" {
// 			return authz.Read, nil
// 		}
// 		return authz.None, nil
// 	})

// 	resolver := newQueryResolver(
// 		database.NewMockDB(),
// 		mockDBStore,
// 		mockLSIFStore,
// 		newCachedCommitChecker(mockGitserverClient),
// 		mockPositionAdjuster,
// 		42,
// 		"deadbeef",
// 		"s1/main.go",
// 		uploads,
// 		newOperations(&observation.TestContext),
// 		checker,
// 		50,
// 		mockSymbolsResolver,
// 	)

// 	ctx := context.Background()
// 	adjustedLocations, err := resolver.Definitions(actor.WithActor(ctx, &actor.Actor{UID: 1}), 10, 20)
// 	if err != nil {
// 		t.Fatalf("unexpected error querying definitions: %s", err)
// 	}

// 	expectedLocations := []AdjustedLocation{
// 		{Dump: uploads[1], Path: "sub2/a.go", AdjustedCommit: "deadbeef", AdjustedRange: testRange1},
// 		{Dump: uploads[1], Path: "sub2/a.go", AdjustedCommit: "deadbeef", AdjustedRange: testRange3},
// 	}
// 	if diff := cmp.Diff(expectedLocations, adjustedLocations); diff != "" {
// 		t.Errorf("unexpected locations (-want +got):\n%s", diff)
// 	}
// }

// func TestDefinitionsRemote(t *testing.T) {
// 	mockDBStore := NewMockDBStore()
// 	mockLSIFStore := NewMockLSIFStore()
// 	mockGitserverClient := NewMockGitserverClient()
// 	mockPositionAdjuster := noopPositionAdjuster()
// 	mockSymbolsResolver := NewMockSymbolsResolver()

// 	dumps := []dbstore.Dump{
// 		{ID: 150, Commit: "deadbeef1", Root: "sub1/"},
// 		{ID: 151, Commit: "deadbeef2", Root: "sub2/"},
// 		{ID: 152, Commit: "deadbeef3", Root: "sub3/"},
// 		{ID: 153, Commit: "deadbeef4", Root: "sub4/"},
// 	}
// 	remoteUploads := storeDumpToSymbolDump(dumps)
// 	mockSymbolsResolver.GetUploadsWithDefinitionsForMonikersFunc.PushReturn(remoteUploads, nil)

// 	// upload #150's commit no longer exists; all others do
// 	mockGitserverClient.CommitsExistFunc.SetDefaultHook(func(ctx context.Context, rcs []gitserver.RepositoryCommit) (exists []bool, _ error) {
// 		for _, rc := range rcs {
// 			exists = append(exists, rc.Commit != "deadbeef1")
// 		}
// 		return
// 	})

// 	monikers := []precise.MonikerData{
// 		{Kind: "import", Scheme: "tsc", Identifier: "padLeft", PackageInformationID: "51"},
// 		{Kind: "export", Scheme: "tsc", Identifier: "pad_left", PackageInformationID: "52"},
// 		{Kind: "import", Scheme: "tsc", Identifier: "pad-left", PackageInformationID: "53"},
// 		{Kind: "import", Scheme: "tsc", Identifier: "left_pad"},
// 	}
// 	mockLSIFStore.MonikersByPositionFunc.PushReturn([][]precise.MonikerData{{monikers[0]}}, nil)
// 	mockLSIFStore.MonikersByPositionFunc.PushReturn([][]precise.MonikerData{{monikers[1]}}, nil)
// 	mockLSIFStore.MonikersByPositionFunc.PushReturn([][]precise.MonikerData{{monikers[2]}}, nil)
// 	mockLSIFStore.MonikersByPositionFunc.PushReturn([][]precise.MonikerData{{monikers[3]}}, nil)

// 	packageInformation1 := precise.PackageInformationData{Name: "leftpad", Version: "0.1.0"}
// 	packageInformation2 := precise.PackageInformationData{Name: "leftpad", Version: "0.2.0"}
// 	mockLSIFStore.PackageInformationFunc.PushReturn(packageInformation1, true, nil)
// 	mockLSIFStore.PackageInformationFunc.PushReturn(packageInformation2, true, nil)

// 	locations := []lsifstore.Location{
// 		{DumpID: 151, Path: "a.go", Range: testRange1},
// 		{DumpID: 151, Path: "b.go", Range: testRange2},
// 		{DumpID: 151, Path: "a.go", Range: testRange3},
// 		{DumpID: 151, Path: "b.go", Range: testRange4},
// 		{DumpID: 151, Path: "c.go", Range: testRange5},
// 	}
// 	mockLSIFStore.BulkMonikerResultsFunc.PushReturn(locations, 0, nil)
// 	mockLSIFStore.BulkMonikerResultsFunc.PushReturn(locations, len(locations), nil)

// 	uploads := []dbstore.Dump{
// 		{ID: 50, Commit: "deadbeef", Root: "sub1/"},
// 		{ID: 51, Commit: "deadbeef", Root: "sub2/"},
// 		{ID: 52, Commit: "deadbeef", Root: "sub3/"},
// 		{ID: 53, Commit: "deadbeef", Root: "sub4/"},
// 	}
// 	resolver := newQueryResolver(
// 		database.NewMockDB(),
// 		mockDBStore,
// 		mockLSIFStore,
// 		newCachedCommitChecker(mockGitserverClient),
// 		mockPositionAdjuster,
// 		42,
// 		"deadbeef",
// 		"s1/main.go",
// 		uploads,
// 		newOperations(&observation.TestContext),
// 		authz.NewMockSubRepoPermissionChecker(),
// 		50,
// 		mockSymbolsResolver,
// 	)
// 	adjustedLocations, err := resolver.Definitions(context.Background(), 10, 20)
// 	if err != nil {
// 		t.Fatalf("unexpected error querying definitions: %s", err)
// 	}

// 	xLocations := []AdjustedLocation{
// 		{Dump: dumps[0], Path: "sub2/a.go", AdjustedCommit: "deadbeef2", AdjustedRange: testRange1},
// 		{Dump: dumps[0], Path: "sub2/b.go", AdjustedCommit: "deadbeef2", AdjustedRange: testRange2},
// 		{Dump: dumps[0], Path: "sub2/a.go", AdjustedCommit: "deadbeef2", AdjustedRange: testRange3},
// 		{Dump: dumps[0], Path: "sub2/b.go", AdjustedCommit: "deadbeef2", AdjustedRange: testRange4},
// 		{Dump: dumps[0], Path: "sub2/c.go", AdjustedCommit: "deadbeef2", AdjustedRange: testRange5},
// 	}
// 	expectedLocations := uploadLocationsToAdjustedLocations(xLocations)
// 	if diff := cmp.Diff(expectedLocations, adjustedLocations); diff != "" {
// 		t.Errorf("unexpected locations (-want +got):\n%s", diff)
// 	}

// 	if history := mockSymbolsResolver.GetUploadsWithDefinitionsForMonikersFunc.History(); len(history) != 1 {
// 		t.Fatalf("unexpected call count for dbstore.DefinitionDump. want=%d have=%d", 1, len(history))
// 	} else {
// 		expectedMonikers := []precise.QualifiedMonikerData{
// 			{MonikerData: monikers[0], PackageInformationData: packageInformation1},
// 			{MonikerData: monikers[2], PackageInformationData: packageInformation2},
// 		}
// 		if diff := cmp.Diff(expectedMonikers, history[0].Arg1); diff != "" {
// 			t.Errorf("unexpected monikers (-want +got):\n%s", diff)
// 		}
// 	}

// 	if history := mockLSIFStore.BulkMonikerResultsFunc.History(); len(history) != 1 {
// 		t.Fatalf("unexpected call count for lsifstore.BulkMonikerResults. want=%d have=%d", 1, len(history))
// 	} else {
// 		if diff := cmp.Diff([]int{151, 152, 153}, history[0].Arg2); diff != "" {
// 			t.Errorf("unexpected ids (-want +got):\n%s", diff)
// 		}

// 		expectedMonikers := []precise.MonikerData{
// 			monikers[0],
// 			monikers[2],
// 		}
// 		if diff := cmp.Diff(expectedMonikers, history[0].Arg3); diff != "" {
// 			t.Errorf("unexpected ids (-want +got):\n%s", diff)
// 		}
// 	}
// }

// func TestDefinitionsRemoteWithSubRepoPermissions(t *testing.T) {
// 	mockDBStore := NewMockDBStore()
// 	mockLSIFStore := NewMockLSIFStore()
// 	mockGitserverClient := NewMockGitserverClient()
// 	mockPositionAdjuster := noopPositionAdjuster()
// 	mockSymbolsResolver := NewMockSymbolsResolver()

// 	dumps := []dbstore.Dump{
// 		{ID: 150, Commit: "deadbeef1", Root: "sub1/"},
// 		{ID: 151, Commit: "deadbeef2", Root: "sub2/"},
// 		{ID: 152, Commit: "deadbeef3", Root: "sub3/"},
// 		{ID: 153, Commit: "deadbeef4", Root: "sub4/"},
// 	}
// 	remoteUploads := storeDumpToSymbolDump(dumps)
// 	mockSymbolsResolver.GetUploadsWithDefinitionsForMonikersFunc.PushReturn(remoteUploads, nil)

// 	// upload #150's commit no longer exists; all others do
// 	mockGitserverClient.CommitsExistFunc.SetDefaultHook(func(ctx context.Context, rcs []gitserver.RepositoryCommit) (exists []bool, _ error) {
// 		for _, rc := range rcs {
// 			exists = append(exists, rc.Commit != "deadbeef1")
// 		}
// 		return
// 	})

// 	monikers := []precise.MonikerData{
// 		{Kind: "import", Scheme: "tsc", Identifier: "padLeft", PackageInformationID: "51"},
// 		{Kind: "export", Scheme: "tsc", Identifier: "pad_left", PackageInformationID: "52"},
// 		{Kind: "import", Scheme: "tsc", Identifier: "pad-left", PackageInformationID: "53"},
// 		{Kind: "import", Scheme: "tsc", Identifier: "left_pad"},
// 	}
// 	mockLSIFStore.MonikersByPositionFunc.PushReturn([][]precise.MonikerData{{monikers[0]}}, nil)
// 	mockLSIFStore.MonikersByPositionFunc.PushReturn([][]precise.MonikerData{{monikers[1]}}, nil)
// 	mockLSIFStore.MonikersByPositionFunc.PushReturn([][]precise.MonikerData{{monikers[2]}}, nil)
// 	mockLSIFStore.MonikersByPositionFunc.PushReturn([][]precise.MonikerData{{monikers[3]}}, nil)

// 	packageInformation1 := precise.PackageInformationData{Name: "leftpad", Version: "0.1.0"}
// 	packageInformation2 := precise.PackageInformationData{Name: "leftpad", Version: "0.2.0"}
// 	mockLSIFStore.PackageInformationFunc.PushReturn(packageInformation1, true, nil)
// 	mockLSIFStore.PackageInformationFunc.PushReturn(packageInformation2, true, nil)

// 	locations := []lsifstore.Location{
// 		{DumpID: 151, Path: "a.go", Range: testRange1},
// 		{DumpID: 151, Path: "b.go", Range: testRange2},
// 		{DumpID: 151, Path: "a.go", Range: testRange3},
// 		{DumpID: 151, Path: "b.go", Range: testRange4},
// 		{DumpID: 151, Path: "c.go", Range: testRange5},
// 	}
// 	mockLSIFStore.BulkMonikerResultsFunc.PushReturn(locations, 0, nil)
// 	mockLSIFStore.BulkMonikerResultsFunc.PushReturn(locations, len(locations), nil)

// 	uploads := []dbstore.Dump{
// 		{ID: 50, Commit: "deadbeef", Root: "sub1/"},
// 		{ID: 51, Commit: "deadbeef", Root: "sub2/"},
// 		{ID: 52, Commit: "deadbeef", Root: "sub3/"},
// 		{ID: 53, Commit: "deadbeef", Root: "sub4/"},
// 	}

// 	// Applying sub-repo permissions
// 	checker := authz.NewMockSubRepoPermissionChecker()

// 	checker.EnabledFunc.SetDefaultHook(func() bool {
// 		return true
// 	})

// 	checker.PermissionsFunc.SetDefaultHook(func(ctx context.Context, i int32, content authz.RepoContent) (authz.Perms, error) {
// 		if content.Path == "sub2/b.go" {
// 			return authz.Read, nil
// 		}
// 		return authz.None, nil
// 	})

// 	resolver := newQueryResolver(
// 		database.NewMockDB(),
// 		mockDBStore,
// 		mockLSIFStore,
// 		newCachedCommitChecker(mockGitserverClient),
// 		mockPositionAdjuster,
// 		42,
// 		"deadbeef",
// 		"s1/main.go",
// 		uploads,
// 		newOperations(&observation.TestContext),
// 		checker,
// 		50,
// 		mockSymbolsResolver,
// 	)

// 	ctx := context.Background()
// 	adjustedLocations, err := resolver.Definitions(actor.WithActor(ctx, &actor.Actor{UID: 1}), 10, 20)
// 	if err != nil {
// 		t.Fatalf("unexpected error querying definitions: %s", err)
// 	}

// 	expectedLocations := []AdjustedLocation{
// 		{Dump: dumps[0], Path: "sub2/b.go", AdjustedCommit: "deadbeef2", AdjustedRange: testRange2},
// 		{Dump: dumps[0], Path: "sub2/b.go", AdjustedCommit: "deadbeef2", AdjustedRange: testRange4},
// 	}
// 	if diff := cmp.Diff(expectedLocations, adjustedLocations); diff != "" {
// 		t.Errorf("unexpected locations (-want +got):\n%s", diff)
// 	}

// 	if history := mockSymbolsResolver.GetUploadsWithDefinitionsForMonikersFunc.History(); len(history) != 1 {
// 		t.Fatalf("unexpected call count for dbstore.DefinitionDump. want=%d have=%d", 1, len(history))
// 	} else {
// 		expectedMonikers := []precise.QualifiedMonikerData{
// 			{MonikerData: monikers[0], PackageInformationData: packageInformation1},
// 			{MonikerData: monikers[2], PackageInformationData: packageInformation2},
// 		}
// 		if diff := cmp.Diff(expectedMonikers, history[0].Arg1); diff != "" {
// 			t.Errorf("unexpected monikers (-want +got):\n%s", diff)
// 		}
// 	}

// 	if history := mockLSIFStore.BulkMonikerResultsFunc.History(); len(history) != 1 {
// 		t.Fatalf("unexpected call count for lsifstore.BulkMonikerResults. want=%d have=%d", 1, len(history))
// 	} else {
// 		if diff := cmp.Diff([]int{151, 152, 153}, history[0].Arg2); diff != "" {
// 			t.Errorf("unexpected ids (-want +got):\n%s", diff)
// 		}

// 		expectedMonikers := []precise.MonikerData{
// 			monikers[0],
// 			monikers[2],
// 		}
// 		if diff := cmp.Diff(expectedMonikers, history[0].Arg3); diff != "" {
// 			t.Errorf("unexpected ids (-want +got):\n%s", diff)
// 		}
// 	}
// }

// func uploadLocationsToAdjustedLocations(location []AdjustedLocation) []shared.UploadLocation {
// 	uploadLocation := make([]shared.UploadLocation, 0, len(location))
// 	for _, loc := range location {
// 		dump := shared.Dump{
// 			ID:                loc.Dump.ID,
// 			Commit:            loc.Dump.Commit,
// 			Root:              loc.Dump.Root,
// 			VisibleAtTip:      loc.Dump.VisibleAtTip,
// 			UploadedAt:        loc.Dump.UploadedAt,
// 			State:             loc.Dump.State,
// 			FailureMessage:    loc.Dump.FailureMessage,
// 			StartedAt:         loc.Dump.StartedAt,
// 			FinishedAt:        loc.Dump.FinishedAt,
// 			ProcessAfter:      loc.Dump.ProcessAfter,
// 			NumResets:         loc.Dump.NumResets,
// 			NumFailures:       loc.Dump.NumFailures,
// 			RepositoryID:      loc.Dump.RepositoryID,
// 			RepositoryName:    loc.Dump.RepositoryName,
// 			Indexer:           loc.Dump.Indexer,
// 			IndexerVersion:    loc.Dump.IndexerVersion,
// 			AssociatedIndexID: loc.Dump.AssociatedIndexID,
// 		}

// 		targetRange := shared.Range{
// 			Start: shared.Position{
// 				Line:      loc.AdjustedRange.Start.Line,
// 				Character: loc.AdjustedRange.Start.Character,
// 			},
// 			End: shared.Position{
// 				Line:      loc.AdjustedRange.End.Line,
// 				Character: loc.AdjustedRange.End.Character,
// 			},
// 		}

// 		uploadLocation = append(uploadLocation, shared.UploadLocation{
// 			Dump:         dump,
// 			Path:         loc.Path,
// 			TargetCommit: loc.AdjustedCommit,
// 			TargetRange:  targetRange,
// 		})
// 	}

// 	return uploadLocation
// }

func storeDumpToSymbolDump(storeDumps []store.Dump) []shared.Dump {
	dumps := make([]shared.Dump, 0, len(storeDumps))
	for _, d := range storeDumps {
		dumps = append(dumps, shared.Dump{
			ID:                d.ID,
			Commit:            d.Commit,
			Root:              d.Root,
			VisibleAtTip:      d.VisibleAtTip,
			UploadedAt:        d.UploadedAt,
			State:             d.State,
			FailureMessage:    d.FailureMessage,
			StartedAt:         d.StartedAt,
			FinishedAt:        d.FinishedAt,
			ProcessAfter:      d.ProcessAfter,
			NumResets:         d.NumResets,
			NumFailures:       d.NumFailures,
			RepositoryID:      d.RepositoryID,
			RepositoryName:    d.RepositoryName,
			Indexer:           d.Indexer,
			IndexerVersion:    d.IndexerVersion,
			AssociatedIndexID: d.AssociatedIndexID,
		})
	}

	return dumps
}
