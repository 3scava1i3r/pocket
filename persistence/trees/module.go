package trees

import (
	"crypto/sha256"
	"fmt"

	"github.com/pokt-network/pocket/persistence/indexer"
	"github.com/pokt-network/pocket/persistence/kvstore"
	"github.com/pokt-network/pocket/shared/modules"
	"github.com/pokt-network/smt"
)

// ErrNoDirectory is thrown when no tree story storage directory is defined.
var ErrNoDirectory = fmt.Errorf("tree store directory must be defined")

// Create sets up treeStore as a module and applies options to it then returns it as a TreeStoreModule
func (*treeStore) Create(bus modules.Bus, options ...modules.TreeStoreOption) (modules.TreeStoreModule, error) {
	m := &treeStore{}

	for _, option := range options {
		option(m)
	}

	m.SetBus(bus)

	if err := m.setupTrees(); err != nil {
		return nil, err
	}

	return m, nil
}

func Create(bus modules.Bus, options ...modules.TreeStoreOption) (modules.TreeStoreModule, error) {
	return new(treeStore).Create(bus, options...)
}

// WithTreeStoreDirectory assigns the path where the tree store
// saves its data.
func WithTreeStoreDirectory(path string) modules.TreeStoreOption {
	return func(m modules.TreeStoreModule) {
		mod, ok := m.(*treeStore)
		if ok {
			mod.treeStoreDir = path
		}
	}
}

func WithTxIndexer(txi indexer.TxIndexer) modules.TreeStoreOption {
	return func(m modules.TreeStoreModule) {
		mod, ok := m.(*treeStore)
		if ok {
			mod.txi = txi
		}
	}
}

func (t *treeStore) setupTrees() error {
	if t.treeStoreDir == "" {
		return ErrNoDirectory
	}
	if t.treeStoreDir == ":memory:" {
		return t.setupInMemory()
	}

	t.merkleTrees = make(map[merkleTree]*smt.SMT, int(numMerkleTrees))
	t.nodeStores = make(map[merkleTree]kvstore.KVStore, int(numMerkleTrees))

	for tree := merkleTree(0); tree < numMerkleTrees; tree++ {
		nodeStore, err := kvstore.NewKVStore(fmt.Sprintf("%s/%s_nodes", t.treeStoreDir, merkleTreeToString[tree]))
		if err != nil {
			return err
		}
		t.nodeStores[tree] = nodeStore
		t.merkleTrees[tree] = smt.NewSparseMerkleTree(nodeStore, sha256.New())
	}

	return nil
}

func (t *treeStore) setupInMemory() error {
	t.merkleTrees = make(map[merkleTree]*smt.SMT, int(numMerkleTrees))
	t.nodeStores = make(map[merkleTree]kvstore.KVStore, int(numMerkleTrees))

	for tree := merkleTree(0); tree < numMerkleTrees; tree++ {
		nodeStore := kvstore.NewMemKVStore() // For testing, `smt.NewSimpleMap()` can be used as well
		t.nodeStores[tree] = nodeStore
		t.merkleTrees[tree] = smt.NewSparseMerkleTree(nodeStore, sha256.New())
	}

	return nil
}
