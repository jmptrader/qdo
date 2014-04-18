package leveldb

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/borgenk/qdo/third_party/github.com/syndtr/goleveldb/leveldb/testutil"
)

func TestLeveldb(t *testing.T) {
	testutil.RunDefer()

	RegisterFailHandler(Fail)
	RunSpecs(t, "Leveldb Suite")

	RegisterTestingT(t)
	testutil.RunDefer("teardown")
}
