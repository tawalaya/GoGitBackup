package backup

import (
	"testing"
	"time"

	"github.com/d5/tengo/v2"
)

func TestGoGitBackup_filter(t *testing.T) {
	owner := Repository{
		Name:       "owner",
		Size:       500,
		CreatedAt:  time.Time{},
		Owner:      true,
		Member:     true,
		Visibility: Public,
	}

	member := Repository{
		Name:       "member",
		Size:       1200,
		CreatedAt:  time.Time{},
		Owner:      false,
		Member:     true,
		Visibility: Public,
	}

	private := Repository{
		Name:       "private",
		Size:       400,
		CreatedAt:  time.Time{},
		Owner:      true,
		Member:     true,
		Visibility: Private,
	}

	test := Repository{
		Name:       "test",
		Size:       1200,
		CreatedAt:  time.Time{},
		Owner:      true,
		Member:     true,
		Visibility: Public,
	}

	garbage := Repository{
		Name:       "garbage",
		Size:       -1,
		CreatedAt:  time.Time{},
		Owner:      false,
		Member:     false,
		Visibility: Internal,
	}

	repos := []Repository{owner, member, private, test, garbage}

	tests := []struct {
		desc     string
		funct    string
		expected []bool
	}{
		{
			"ignore all I don't own",
			"r := !owner",
			[]bool{false, true, false, false, true},
		},
		{
			desc:     "members",
			funct:    "r := !member",
			expected: []bool{false, false, false, false, true},
		},
		{
			desc:     "small and owned",
			funct:    "r := !owner || size > 600",
			expected: []bool{false, true, false, true, true},
		},
		{
			desc:     "expect all to be false",
			funct:    "r := false",
			expected: []bool{false, false, false, false, false},
		},
	}

	for _, test := range tests {
		function := tengo.NewScript([]byte(test.funct))
		for i, r := range repos {
			result := apply(function, r)
			if result != test.expected[i] {
				t.Fatal("failed", test.desc, "for", r.Name, "got", result, "expected", test.expected[i])
			}
		}
	}

}
