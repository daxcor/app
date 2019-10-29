package image

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"

	"gotest.tools/assert"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
)

type bundleStoreStubForListCmd struct {
	refMap map[reference.Reference]*bundle.Bundle
	// in order to keep the reference in the same order between tests
	refList []reference.Reference
}

func (b *bundleStoreStubForListCmd) Store(ref reference.Reference, bndle *bundle.Bundle) (reference.Digested, error) {
	b.refMap[ref] = bndle
	b.refList = append(b.refList, ref)
	return store.FromBundle(bndle)
}

func (b *bundleStoreStubForListCmd) Read(ref reference.Reference) (*bundle.Bundle, error) {
	bndl, ok := b.refMap[ref]
	if ok {
		return bndl, nil
	}
	return nil, fmt.Errorf("Bundle not found")
}

func (b *bundleStoreStubForListCmd) List() ([]reference.Reference, error) {
	return b.refList, nil
}

func (b *bundleStoreStubForListCmd) Remove(ref reference.Reference) error {
	return nil
}

func (b *bundleStoreStubForListCmd) LookUp(refOrID string) (reference.Reference, error) {
	return nil, nil
}

func TestListCmd(t *testing.T) {
	ref, err := store.FromString("a855ac937f2ed375ba4396bbc49c4093e124da933acd2713fb9bc17d7562a087")
	assert.NilError(t, err)
	refs := []reference.Reference{
		parseReference(t, "foo/bar@sha256:b59492bb814012ca3d2ce0b6728242d96b4af41687cc82166a4b5d7f2d9fb865"),
		parseReference(t, "foo/bar:1.0"),
		ref,
	}
	bundles := []bundle.Bundle{
		{
			Name: "Digested App",
		},
		{
			Version:       "1.0.0",
			SchemaVersion: "1.0.0",
			Name:          "Foo App",
		},
		{
			Name: "Quiet App",
		},
	}

	testCases := []struct {
		name           string
		expectedOutput string
		options        imageListOption
	}{
		{
			name: "TestList",
			expectedOutput: `REPOSITORY                                                       TAG    APP IMAGE ID APP NAME
foo/bar                                                          <none> 3f825b2d0657 Digested App
foo/bar                                                          1.0    9aae408ee04f Foo App
a855ac937f2ed375ba4396bbc49c4093e124da933acd2713fb9bc17d7562a087 <none> a855ac937f2e Quiet App
`,
			options: imageListOption{},
		},
		{
			name: "TestListWithDigests",
			expectedOutput: `REPOSITORY                                                       TAG    DIGEST                                                                  APP IMAGE ID APP NAME
foo/bar                                                          <none> sha256:b59492bb814012ca3d2ce0b6728242d96b4af41687cc82166a4b5d7f2d9fb865 3f825b2d0657 Digested App
foo/bar                                                          1.0    <none>                                                                  9aae408ee04f Foo App
a855ac937f2ed375ba4396bbc49c4093e124da933acd2713fb9bc17d7562a087 <none> sha256:a855ac937f2ed375ba4396bbc49c4093e124da933acd2713fb9bc17d7562a087 a855ac937f2e Quiet App
`,
			options: imageListOption{digests: true},
		},
		{
			name: "TestListWithQuiet",
			expectedOutput: `3f825b2d0657
9aae408ee04f
a855ac937f2e
`,
			options: imageListOption{quiet: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testRunList(t, refs, bundles, tc.options, tc.expectedOutput)
		})
	}
}

func parseReference(t *testing.T, s string) reference.Reference {
	ref, err := reference.Parse(s)
	assert.NilError(t, err)
	return ref
}

func testRunList(t *testing.T, refs []reference.Reference, bundles []bundle.Bundle, options imageListOption, expectedOutput string) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	dockerCli, err := command.NewDockerCli(command.WithOutputStream(w))
	assert.NilError(t, err)
	bundleStore := &bundleStoreStubForListCmd{
		refMap:  make(map[reference.Reference]*bundle.Bundle),
		refList: []reference.Reference{},
	}
	for i, ref := range refs {
		_, err = bundleStore.Store(ref, &bundles[i])
		assert.NilError(t, err)
	}
	err = runList(dockerCli, options, bundleStore)
	assert.NilError(t, err)
	w.Flush()
	assert.Equal(t, buf.String(), expectedOutput)
}
