package querybuilder

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hexops/autogold"

	"github.com/sourcegraph/sourcegraph/internal/search/query"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

func TestWithDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		defaults query.Parameters
	}{
		{
			name:     "no defaults",
			input:    "repo:myrepo testquery",
			want:     "repo:myrepo testquery",
			defaults: []query.Parameter{},
		},
		{
			name:     "no defaults with fork archived",
			input:    "repo:myrepo testquery fork:no archived:no",
			want:     "repo:myrepo fork:no archived:no testquery",
			defaults: []query.Parameter{},
		},
		{
			name:     "no defaults with patterntype",
			input:    "repo:myrepo testquery patterntype:standard",
			want:     "repo:myrepo patterntype:standard testquery",
			defaults: []query.Parameter{},
		},
		{
			name:  "default archived",
			input: "repo:myrepo testquery fork:no",
			want:  "archived:yes repo:myrepo fork:no testquery",
			defaults: []query.Parameter{{
				Field:      query.FieldArchived,
				Value:      string(query.Yes),
				Negated:    false,
				Annotation: query.Annotation{},
			}},
		},
		{
			name:  "default fork and archived",
			input: "repo:myrepo testquery",
			want:  "archived:no fork:no repo:myrepo testquery",
			defaults: []query.Parameter{{
				Field:      query.FieldArchived,
				Value:      string(query.No),
				Negated:    false,
				Annotation: query.Annotation{},
			}, {
				Field:      query.FieldFork,
				Value:      string(query.No),
				Negated:    false,
				Annotation: query.Annotation{},
			}},
		},
		{
			name:  "default patterntype",
			input: "repo:myrepo testquery",
			want:  "patterntype:literal repo:myrepo testquery",
			defaults: []query.Parameter{{
				Field:      query.FieldPatternType,
				Value:      "literal",
				Negated:    false,
				Annotation: query.Annotation{},
			}},
		},
		{
			name:  "default patterntype does not override",
			input: "patterntype:standard repo:myrepo testquery",
			want:  "patterntype:standard repo:myrepo testquery",
			defaults: []query.Parameter{{
				Field:      query.FieldPatternType,
				Value:      "literal",
				Negated:    false,
				Annotation: query.Annotation{},
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := withDefaults(BasicQuery(test.input), test.defaults)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Fatalf("%s failed (want/got): %s", test.name, diff)
			}
		})
	}
}

func TestWithDefaultsPatternTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		defaults query.Parameters
	}{
		{
			// It's worth noting that we always append patterntype:regexp to capture group queries.
			name:     "regexp query without patterntype",
			input:    `file:go\.mod$ go\s*(\d\.\d+)`,
			want:     `file:go\.mod$ go\s*(\d\.\d+)`,
			defaults: []query.Parameter{},
		},
		{
			name:     "regexp query with patterntype",
			input:    `file:go\.mod$ go\s*(\d\.\d+) patterntype:regexp`,
			want:     `file:go\.mod$ patterntype:regexp go\s*(\d\.\d+)`,
			defaults: []query.Parameter{},
		},
		{
			name:     "literal query without patterntype",
			input:    `package search`,
			want:     `package search`,
			defaults: []query.Parameter{},
		},
		{
			name:     "literal query with patterntype",
			input:    `package search patterntype:literal`,
			want:     `patterntype:literal package search`,
			defaults: []query.Parameter{},
		},
		{
			name:     "literal query with quotes without patterntype",
			input:    `"license": "A`,
			want:     `"license": "A`,
			defaults: []query.Parameter{},
		},
		{
			name:     "literal query with quotes with patterntype",
			input:    `"license": "A patterntype:literal`,
			want:     `patterntype:literal "license": "A`,
			defaults: []query.Parameter{},
		},
		{
			name:     "structural query without patterntype",
			input:    `TODO(...)`,
			want:     `TODO(...)`,
			defaults: []query.Parameter{},
		},
		{
			name:     "structural query with patterntype",
			input:    `TODO(...) patterntype:structural`,
			want:     `patterntype:structural TODO(...)`,
			defaults: []query.Parameter{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := withDefaults(BasicQuery(test.input), test.defaults)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Fatalf("%s failed (want/got): %s", test.name, diff)
			}
		})
	}
}

func TestMultiRepoQuery(t *testing.T) {
	tests := []struct {
		name     string
		repos    []string
		want     string
		defaults query.Parameters
	}{
		{
			name:     "single repo",
			repos:    []string{"repo1"},
			want:     `count:99999999 testquery repo:^(repo1)$`,
			defaults: []query.Parameter{},
		},
		{
			name:  "multiple repo",
			repos: []string{"repo1", "repo2"},
			want:  `archived:no fork:no count:99999999 testquery repo:^(repo1|repo2)$`,
			defaults: []query.Parameter{{
				Field:      query.FieldArchived,
				Value:      string(query.No),
				Negated:    false,
				Annotation: query.Annotation{},
			}, {
				Field:      query.FieldFork,
				Value:      string(query.No),
				Negated:    false,
				Annotation: query.Annotation{},
			}},
		},
		{
			name:  "multiple repo",
			repos: []string{"github.com/myrepos/repo1", "github.com/myrepos/repo2"},
			want:  `archived:no fork:no count:99999999 testquery repo:^(github\.com/myrepos/repo1|github\.com/myrepos/repo2)$`,
			defaults: []query.Parameter{{
				Field:      query.FieldArchived,
				Value:      string(query.No),
				Negated:    false,
				Annotation: query.Annotation{},
			}, {
				Field:      query.FieldFork,
				Value:      string(query.No),
				Negated:    false,
				Annotation: query.Annotation{},
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := MultiRepoQuery("testquery", test.repos, test.defaults)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Fatalf("%s failed (want/got): %s", test.name, diff)
			}
		})
	}
}

func TestDefaults(t *testing.T) {
	tests := []struct {
		name  string
		input bool
		want  query.Parameters
	}{
		{
			name:  "all repos",
			input: true,
			want: query.Parameters{{
				Field:      query.FieldFork,
				Value:      string(query.No),
				Negated:    false,
				Annotation: query.Annotation{},
			}, {
				Field:      query.FieldArchived,
				Value:      string(query.No),
				Negated:    false,
				Annotation: query.Annotation{},
			}, {
				Field:      query.FieldPatternType,
				Value:      "literal",
				Negated:    false,
				Annotation: query.Annotation{},
			}},
		},
		{
			name:  "some repos",
			input: false,
			want: query.Parameters{{
				Field:      query.FieldFork,
				Value:      string(query.Yes),
				Negated:    false,
				Annotation: query.Annotation{},
			}, {
				Field:      query.FieldArchived,
				Value:      string(query.Yes),
				Negated:    false,
				Annotation: query.Annotation{},
			}, {
				Field:      query.FieldPatternType,
				Value:      "literal",
				Negated:    false,
				Annotation: query.Annotation{},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := CodeInsightsQueryDefaults(test.input)

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Fatalf("%s failed (want/got): %s", test.name, diff)
			}
		})
	}
}

func TestComputeInsightCommandQuery(t *testing.T) {
	tests := []struct {
		name       string
		inputQuery string
		mapType    MapType
		want       string
	}{
		{
			name:       "verify archive fork map to lang",
			inputQuery: "repo:abc123@12346f fork:yes archived:yes findme",
			mapType:    Lang,
			want:       "repo:abc123@12346f fork:yes archived:yes content:output.extra(findme -> $lang)",
		}, {
			name:       "verify archive fork map to repo",
			inputQuery: "repo:abc123@12346f fork:yes archived:yes findme",
			mapType:    Repo,
			want:       "repo:abc123@12346f fork:yes archived:yes content:output.extra(findme -> $repo)",
		}, {
			name:       "verify archive fork map to path",
			inputQuery: "repo:abc123@12346f fork:yes archived:yes findme",
			mapType:    Path,
			want:       "repo:abc123@12346f fork:yes archived:yes content:output.extra(findme -> $path)",
		}, {
			name:       "verify archive fork map to author",
			inputQuery: "repo:abc123@12346f fork:yes archived:yes findme",
			mapType:    Author,
			want:       "repo:abc123@12346f fork:yes archived:yes content:output.extra(findme -> $author)",
		}, {
			name:       "verify archive fork map to date",
			inputQuery: "repo:abc123@12346f fork:yes archived:yes findme",
			mapType:    Date,
			want:       "repo:abc123@12346f fork:yes archived:yes content:output.extra(findme -> $date)",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ComputeInsightCommandQuery(BasicQuery(test.inputQuery), test.mapType)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(test.want, string(got)); diff != "" {
				t.Errorf("%s failed (want/got): %s", test.name, diff)
			}
		})
	}
}

func TestIsSingleRepoQuery(t *testing.T) {

	tests := []struct {
		name       string
		inputQuery string
		mapType    MapType
		want       bool
	}{
		{
			name:       "repo as simple text string",
			inputQuery: "repo:abc123@12346f fork:yes archived:yes findme",
			mapType:    Lang,
			want:       false,
		},
		{
			name:       "repo contains path",
			inputQuery: "repo:contains.path(CHANGELOG) TEST",
			mapType:    Lang,
			want:       false,
		},
		{
			name:       "repo or",
			inputQuery: "repo:^(repo1|repo2)$ test",
			mapType:    Lang,
			want:       false,
		},
		{
			name:       "single repo with revision specified",
			inputQuery: `repo:^github\.com/sgtest/java-langserver$@v1 test`,
			mapType:    Lang,
			want:       true,
		},
		{
			name:       "single repo",
			inputQuery: `repo:^github\.com/sgtest/java-langserver$ test`,
			mapType:    Lang,
			want:       true,
		},
		{
			name:       "query without repo filter",
			inputQuery: `test`,
			mapType:    Lang,
			want:       false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := IsSingleRepoQuery(BasicQuery(test.inputQuery))
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("%s failed (want/got): %s", test.name, diff)
			}

		})
	}
}

func TestIsSingleRepoQueryMultipleSteps(t *testing.T) {

	tests := []struct {
		name       string
		inputQuery string
		mapType    MapType
		want       error
	}{
		{
			name:       "2 step query different repos",
			inputQuery: `(repo:^github\.com/sourcegraph/sourcegraph$ OR repo:^github\.com/sourcegraph-testing/zap$) test`,
			mapType:    Lang,
			want:       QueryNotSupported,
		},
		{
			name:       "2 step query same repo",
			inputQuery: `(repo:^github\.com/sourcegraph/sourcegraph$ test) OR (repo:^github\.com/sourcegraph/sourcegraph$ todo)`,
			mapType:    Lang,
			want:       QueryNotSupported,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := IsSingleRepoQuery(BasicQuery(test.inputQuery))
			if !errors.Is(err, test.want) {
				t.Error(err)
			}
			if diff := cmp.Diff(false, got); diff != "" {
				t.Errorf("%s failed (want/got): %s", test.name, diff)
			}

		})
	}
}

func TestAggregationQuery(t *testing.T) {

	tests := []struct {
		name       string
		inputQuery string
		count      string
		want       autogold.Value
	}{
		{
			inputQuery: `test`,
			count:      "all",
			want:       autogold.Want("basic query", BasicQuery("count:all timeout:2s test")),
		},
		{
			inputQuery: `(repo:^github\.com/sourcegraph/sourcegraph$ test) OR (repo:^github\.com/sourcegraph/sourcegraph$ todo)`,
			count:      "all",
			want:       autogold.Want("multiplan query", BasicQuery("(repo:^github\\.com/sourcegraph/sourcegraph$ count:all timeout:2s test OR repo:^github\\.com/sourcegraph/sourcegraph$ count:all timeout:2s todo)")),
		},
		{
			inputQuery: `(repo:^github\.com/sourcegraph/sourcegraph$ test) OR (repo:^github\.com/sourcegraph/sourcegraph$ todo) count:2000`,
			count:      "all",
			want:       autogold.Want("multiplan query overwrite", BasicQuery("(repo:^github\\.com/sourcegraph/sourcegraph$ count:all timeout:2s test OR repo:^github\\.com/sourcegraph/sourcegraph$ count:all timeout:2s todo)")),
		},
		{
			inputQuery: `test count:1000`,
			count:      "all",
			want:       autogold.Want("overwrite existing", BasicQuery("count:all timeout:2s test")),
		},
		{
			inputQuery: `test count:1000`,
			count:      "50000",
			want:       autogold.Want("overwrite existing", BasicQuery("count:50000 timeout:2s test")),
		},
	}
	for _, test := range tests {
		t.Run(test.want.Name(), func(t *testing.T) {
			got, _ := AggregationQuery(BasicQuery(test.inputQuery), 2, test.count)
			test.want.Equal(t, got)

		})
	}
}

func Test_addAuthorFilter(t *testing.T) {
	tests := []struct {
		input  string
		author string
		want   autogold.Value
	}{
		{
			input:  "myquery repo:myrepo type:commit",
			author: "santa",
			want:   autogold.Want("no initial author field in commit search", BasicQuery("repo:myrepo type:commit author:^santa$ myquery")),
		},
		{
			input:  "myquery repo:myrepo type:commit",
			author: "xtreme[username]",
			want:   autogold.Want("ensure author is escaped", BasicQuery("repo:myrepo type:commit author:^xtreme\\[username\\]$ myquery")),
		},
		{
			input:  "myquery repo:myrepo type:commit author:claus",
			author: "santa",
			want:   autogold.Want("one initial author field in commit search", BasicQuery("repo:myrepo type:commit author:claus author:^santa$ myquery")),
		},
		{
			input:  "myquery repo:myrepo type:diff",
			author: "santa",
			want:   autogold.Want("no initial author field in diff search", BasicQuery("repo:myrepo type:diff author:^santa$ myquery")),
		},
		{
			input:  "myquery repo:myrepo type:diff author:claus",
			author: "santa",
			want:   autogold.Want("one initial author field in diff search", BasicQuery("repo:myrepo type:diff author:claus author:^santa$ myquery")),
		},
		{
			input:  "myquery repo:myrepo type:file author:claus",
			author: "santa",
			want:   autogold.Want("invalid adding to file search - should error", "your query contains the field 'author', which requires type:commit or type:diff in the query"),
		},
		{
			input:  "myquery repo:myrepo type:repo",
			author: "santa",
			want:   autogold.Want("invalid adding to repo search - should return input", BasicQuery("repo:myrepo type:repo myquery")),
		},
		{
			input:  "(myquery repo:myrepo type:repo) or (type:diff repo:asdf findme)",
			author: "santa",
			want:   autogold.Want("compound query where one side is author and one side is repo", BasicQuery("(repo:myrepo type:repo myquery OR type:diff repo:asdf author:^santa$ findme)")),
		},
		{
			input:  "insights type:commit",
			author: "Santa Claus",
			want:   autogold.Want("author with whitespace in name", BasicQuery("type:commit author:(^Santa Claus$) insights")),
		},
	}
	for _, test := range tests {
		t.Run(test.want.Name(), func(t *testing.T) {
			got, err := AddAuthorFilter(BasicQuery(test.input), test.author)
			if err != nil {
				test.want.Equal(t, err.Error())
			} else {
				test.want.Equal(t, got)
			}
		})
	}
}

func Test_addRepoFilter(t *testing.T) {
	tests := []struct {
		input string
		repo  string
		want  autogold.Value
	}{
		{
			input: "myquery",
			repo:  "github.com/sourcegraph/sourcegraph",
			want:  autogold.Want("no initial repo filter", BasicQuery("repo:^github\\.com/sourcegraph/sourcegraph$ myquery")),
		},
		{
			input: "myquery repo:supergreat",
			repo:  "github.com/sourcegraph/sourcegraph",
			want:  autogold.Want("one initial repo filter", BasicQuery("repo:supergreat repo:^github\\.com/sourcegraph/sourcegraph$ myquery")),
		},
		{
			input: "(myquery repo:supergreat) or (big repo:asdf)",
			repo:  "github.com/sourcegraph/sourcegraph",
			want:  autogold.Want("compound query adding repo", BasicQuery("(repo:supergreat repo:^github\\.com/sourcegraph/sourcegraph$ myquery OR repo:asdf repo:^github\\.com/sourcegraph/sourcegraph$ big)")),
		},
	}
	for _, test := range tests {
		t.Run(test.want.Name(), func(t *testing.T) {
			got, err := AddRepoFilter(BasicQuery(test.input), test.repo)
			if err != nil {
				test.want.Equal(t, err.Error())
			} else {
				test.want.Equal(t, got)
			}
		})
	}
}

func Test_addFileFilter(t *testing.T) {
	tests := []struct {
		input string
		file  string
		want  autogold.Value
	}{
		{
			input: "myquery",
			file:  "some/directory/file.md",
			want:  autogold.Want("no initial repo filter", BasicQuery("file:^some/directory/file\\.md$ myquery")),
		},
		{
			input: "myquery repo:supergreat",
			file:  "some/directory/file.md",
			want:  autogold.Want("one initial repo filter", BasicQuery("repo:supergreat file:^some/directory/file\\.md$ myquery")),
		},
		{
			input: "(myquery repo:supergreat file:abcdef) or (big repo:asdf)",
			file:  "some/directory/file.md",
			want:  autogold.Want("compound query adding file", BasicQuery("(repo:supergreat file:abcdef file:^some/directory/file\\.md$ myquery OR repo:asdf file:^some/directory/file\\.md$ big)")),
		},
	}
	for _, test := range tests {
		t.Run(test.want.Name(), func(t *testing.T) {
			got, err := AddFileFilter(BasicQuery(test.input), test.file)
			if err != nil {
				test.want.Equal(t, err.Error())
			} else {
				test.want.Equal(t, got)
			}
		})
	}
}

func TestRepositoryScopeQuery(t *testing.T) {
	tests := []struct {
		input string
		want  autogold.Value
	}{
		{
			"repo:sourcegraph",
			autogold.Want("basic query", BasicQuery("fork:yes archived:yes count:all repo:sourcegraph")),
		},
		{
			"repo:s or repo:l",
			autogold.Want("compound query", BasicQuery("(fork:yes archived:yes count:all repo:s OR fork:yes archived:yes count:all repo:l)")),
		},
		{
			"repo:a fork:n",
			autogold.Want("overwrites fork: values", BasicQuery("fork:yes archived:yes count:all repo:a")),
		},
	}
	for _, test := range tests {
		t.Run(test.want.Name(), func(t *testing.T) {
			got, err := RepositoryScopeQuery(test.input)
			if err != nil {
				test.want.Equal(t, err.Error())
			} else {
				test.want.Equal(t, got)
			}
		})
	}
}

func TestWithCount(t *testing.T) {
	tests := []struct {
		input BasicQuery
		want  autogold.Value
	}{
		{
			BasicQuery("repo:sourcegraph"),
			autogold.Want("adds count", BasicQuery("repo:sourcegraph count:99")),
		},
		{
			BasicQuery("repo:s or repo:l"),
			autogold.Want("compound query", BasicQuery("(repo:s count:99 OR repo:l count:99)")),
		},
		{
			BasicQuery("repo:a count:1"),
			autogold.Want("overwrites count values", BasicQuery("repo:a count:99")),
		},
		{
			BasicQuery("repo:a count:all"),
			autogold.Want("overwrites count all", BasicQuery("repo:a count:99")),
		},
	}
	for _, test := range tests {
		t.Run(test.want.Name(), func(t *testing.T) {
			got, err := test.input.WithCount("99")
			if err != nil {
				test.want.Equal(t, err.Error())
			} else {
				test.want.Equal(t, got)
			}
		})
	}
}

func TestMakeQueryWithRepoFilters(t *testing.T) {
	tests := []struct {
		repos string
		query string
		want  autogold.Value
	}{
		{
			"repo:sourcegraph",
			"insights",
			autogold.Want("simple repo with simple query", BasicQuery("repo:sourcegraph fork:no archived:no patterntype:literal count:99999999 insights")),
		},
		{
			"repo:sourcegraph or repo:handbook",
			"insights",
			autogold.Want("compound repo with simple query", BasicQuery("(repo:sourcegraph OR repo:handbook) fork:no archived:no patterntype:literal count:99999999 insights")),
		},
		{
			"repo:sourcegraph",
			"insights or todo",
			autogold.Want("simple repo with compound query", BasicQuery("repo:sourcegraph (fork:no archived:no patterntype:literal insights OR fork:no archived:no patterntype:literal count:99999999 todo)")),
		},
		{
			"repo:sourcegraph or repo:has.file(content:sourcegraph)",
			"insights or todo",
			autogold.Want("compound repo with compound query", BasicQuery("(repo:sourcegraph OR repo:has.file(content:sourcegraph)) (fork:no archived:no patterntype:literal insights OR fork:no archived:no patterntype:literal count:99999999 todo)")),
		},
		{
			"repo:test or repo:handbook",
			"insights fork:yes",
			autogold.Want("compound repo with fork:yes query", BasicQuery("(repo:test OR repo:handbook) archived:no patterntype:literal fork:yes count:99999999 insights")),
		},
		{
			"repo:regex",
			`I\slove patterntype:regexp`,
			autogold.Want("regexp query", BasicQuery(`repo:regex fork:no archived:no patterntype:regexp count:99999999 I\slove`)),
		},
	}
	for _, test := range tests {
		t.Run(test.want.Name(), func(t *testing.T) {
			got, err := MakeQueryWithRepoFilters(test.repos, BasicQuery(test.query), true, CodeInsightsQueryDefaults(true)...)
			if err != nil {
				test.want.Equal(t, err.Error())
			} else {
				test.want.Equal(t, got)
			}
		})
	}
}

func TestPointDiffQuery(t *testing.T) {

	before := time.Date(2022, time.February, 1, 1, 1, 0, 0, time.UTC)
	after := time.Date(2022, time.January, 1, 1, 1, 0, 0, time.UTC)
	repoSearch := "repo:.*"
	complexRepoSearch := "repo:sourcegraph or repo:about"

	tests := []struct {
		opts PointDiffQueryOpts
		want autogold.Value
	}{
		{
			PointDiffQueryOpts{
				Before:             before,
				After:              &after,
				FilterRepoIncludes: []string{"repo1", "repo2"},
				SearchQuery:        BasicQuery("insights"),
			},
			autogold.Want("multiple includes or together", BasicQuery("repo:repo1|repo2 after:2022-01-01T01:01:00Z before:2022-02-01T01:01:00Z type:diff insights")),
		},
		{
			PointDiffQueryOpts{
				Before:             before,
				After:              &after,
				FilterRepoExcludes: []string{"repo1", "repo2"},
				SearchQuery:        BasicQuery("insights"),
			},
			autogold.Want("multiple excludes or together", BasicQuery("-repo:repo1|repo2 after:2022-01-01T01:01:00Z before:2022-02-01T01:01:00Z type:diff insights")),
		},
		{
			PointDiffQueryOpts{
				Before:      before,
				After:       &after,
				RepoList:    []string{"github.com/sourcegraph/sourcegraph", "github.com/sourcegraph/about"},
				SearchQuery: BasicQuery("insights"),
			},
			autogold.Want("repo list escaped and or together", BasicQuery("repo:^(github\\.com/sourcegraph/sourcegraph|github\\.com/sourcegraph/about)$ after:2022-01-01T01:01:00Z before:2022-02-01T01:01:00Z type:diff insights")),
		},
		{
			PointDiffQueryOpts{
				Before:      before,
				After:       &after,
				RepoSearch:  &repoSearch,
				SearchQuery: BasicQuery("insights"),
			},
			autogold.Want("repo search added", BasicQuery("repo:.* after:2022-01-01T01:01:00Z before:2022-02-01T01:01:00Z type:diff insights")),
		},
		{
			PointDiffQueryOpts{
				Before:             before,
				After:              &after,
				FilterRepoIncludes: []string{"repoa", "repob"},
				FilterRepoExcludes: []string{"repo1", "repo2"},
				SearchQuery:        BasicQuery("insights"),
			},
			autogold.Want("include and excluded can be used together", BasicQuery("repo:repoa|repob -repo:repo1|repo2 after:2022-01-01T01:01:00Z before:2022-02-01T01:01:00Z type:diff insights")),
		},
		{
			PointDiffQueryOpts{
				Before:      before,
				SearchQuery: BasicQuery("insights"),
			},
			autogold.Want("after isn't needed", BasicQuery("before:2022-02-01T01:01:00Z type:diff insights")),
		},
		{
			PointDiffQueryOpts{
				Before:             before,
				After:              &after,
				FilterRepoIncludes: []string{"sourcegraph", "about"},
				FilterRepoExcludes: []string{"mega", "multierr"},
				SearchQuery:        BasicQuery("insights or worker"),
				RepoSearch:         &complexRepoSearch,
			},
			autogold.Want("compound repo search and include/exclude", BasicQuery("(repo:sourcegraph OR repo:about) repo:sourcegraph|about -repo:mega|multierr after:2022-01-01T01:01:00Z before:2022-02-01T01:01:00Z type:diff (insights OR worker)")),
		},
		{
			PointDiffQueryOpts{
				Before:             before,
				After:              &after,
				FilterRepoIncludes: []string{"repo1|repo2"},
				SearchQuery:        BasicQuery("insights"),
			},
			autogold.Want("regex in include", BasicQuery("repo:repo1|repo2 after:2022-01-01T01:01:00Z before:2022-02-01T01:01:00Z type:diff insights")),
		},
	}
	for _, test := range tests {
		t.Run(test.want.Name(), func(t *testing.T) {
			got, err := PointDiffQuery(test.opts)
			if err != nil {
				test.want.Equal(t, err.Error())
			} else {
				test.want.Equal(t, got)
			}
		})
	}
}
