package kapp

import (
	"github.com/stretchr/testify/assert"
	"github.com/sugarkube/sugarkube/internal/pkg/acquirer"
	"gopkg.in/yaml.v2"
	"testing"
)

func TestParseManifest(t *testing.T) {
	tests := []struct {
		name                 string
		desc                 string
		input                string
		inputShouldBePresent bool
		expectValues         []Kapp
		expectedError        bool
	}{
		{
			name: "good_parse",
			desc: "check parsing acceptable input works",
			input: `
present:
  example1:
    sources:
    - uri: git@github.com:exampleA/repoA.git
      branch: branchA
      path: example/pathA
    - uri: git@github.com:exampleB/repoB.git
      branch: branchB
      path: example/pathB
`,
			expectValues: []Kapp{
				{
					id:              "example1",
					shouldBePresent: true,
					sources: []acquirer.Acquirer{
						acquirer.NewGitAcquirer("git@github.com:exampleA/repoA.git",
							"branchA", "example/pathA"),
						acquirer.NewGitAcquirer("git@github.com:exampleB/repoB.git",
							"branchB", "example/pathB"),
					},
				},
			},
			expectedError: false,
		},
	}

	for _, test := range tests {
		inputYaml := map[string]interface{}{}
		err := yaml.Unmarshal([]byte(test.input), inputYaml)
		assert.Nil(t, err)

		result, err := parseManifestYaml(inputYaml)
		if test.expectedError {
			assert.NotNil(t, err)
			assert.Nil(t, result)
		} else {
			assert.Equal(t, test.expectValues, result, "unexpected conversion result for %s", test.name)
			assert.Nil(t, err)
		}
	}
}