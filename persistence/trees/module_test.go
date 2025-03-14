package trees_test

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/pocket/internal/testutil"
	"github.com/pokt-network/pocket/p2p/providers/peerstore_provider"
	"github.com/pokt-network/pocket/persistence/trees"
	"github.com/pokt-network/pocket/runtime"
	"github.com/pokt-network/pocket/runtime/genesis"
	"github.com/pokt-network/pocket/runtime/test_artifacts"
	coreTypes "github.com/pokt-network/pocket/shared/core/types"
	cryptoPocket "github.com/pokt-network/pocket/shared/crypto"
	"github.com/pokt-network/pocket/shared/modules"
	mockModules "github.com/pokt-network/pocket/shared/modules/mocks"
)

const (
	serviceURLFormat = "node%d.consensus:42069"
)

func TestTreeStore_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRuntimeMgr := mockModules.NewMockRuntimeMgr(ctrl)
	mockBus := createMockBus(t, mockRuntimeMgr)
	genesisStateMock := createMockGenesisState(nil)
	persistenceMock := preparePersistenceMock(t, mockBus, genesisStateMock)

	mockBus.EXPECT().
		GetPersistenceModule().
		Return(persistenceMock).
		AnyTimes()
	persistenceMock.EXPECT().
		GetBus().
		AnyTimes().
		Return(mockBus)
	persistenceMock.EXPECT().
		NewRWContext(int64(0)).
		AnyTimes()
	persistenceMock.EXPECT().
		GetTxIndexer().
		AnyTimes()

	treemod, err := trees.Create(mockBus, trees.WithTreeStoreDirectory(":memory:"))
	assert.NoError(t, err)

	got := treemod.GetBus()
	assert.Equal(t, got, mockBus)

	// root hash should be empty for empty tree
	root, ns := treemod.GetTree(trees.TransactionsTreeName)
	require.Equal(t, root, make([]byte, 32))

	// nodestore should have no values in it
	keys, vals, err := ns.GetAll(nil, false)
	require.NoError(t, err)
	require.Empty(t, keys, vals)
}

func TestTreeStore_DebugClearAll(t *testing.T) {
	// TODO: Write test case for the DebugClearAll method
	t.Skip("TODO: Write test case for DebugClearAll method")
}

// createMockGenesisState configures and returns a mocked GenesisState
func createMockGenesisState(valKeys []cryptoPocket.PrivateKey) *genesis.GenesisState {
	genesisState := new(genesis.GenesisState)
	validators := make([]*coreTypes.Actor, len(valKeys))
	for i, valKey := range valKeys {
		addr := valKey.Address().String()
		mockActor := &coreTypes.Actor{
			ActorType:       coreTypes.ActorType_ACTOR_TYPE_VAL,
			Address:         addr,
			PublicKey:       valKey.PublicKey().String(),
			ServiceUrl:      validatorId(i + 1),
			StakedAmount:    test_artifacts.DefaultStakeAmountString,
			PausedHeight:    int64(0),
			UnstakingHeight: int64(0),
			Output:          addr,
		}
		validators[i] = mockActor
	}
	genesisState.Validators = validators

	return genesisState
}

// Persistence mock - only needed for validatorMap access
func preparePersistenceMock(t *testing.T, busMock *mockModules.MockBus, genesisState *genesis.GenesisState) *mockModules.MockPersistenceModule {
	ctrl := gomock.NewController(t)

	persistenceModuleMock := mockModules.NewMockPersistenceModule(ctrl)
	readCtxMock := mockModules.NewMockPersistenceReadContext(ctrl)

	readCtxMock.EXPECT().
		GetAllValidators(gomock.Any()).
		Return(genesisState.GetValidators(), nil).AnyTimes()
	readCtxMock.EXPECT().
		GetAllStakedActors(gomock.Any()).
		DoAndReturn(func(height int64) ([]*coreTypes.Actor, error) {
			return testutil.Concatenate[*coreTypes.Actor](
				genesisState.GetValidators(),
				genesisState.GetServicers(),
				genesisState.GetFishermen(),
				genesisState.GetApplications(),
			), nil
		}).
		AnyTimes()
	persistenceModuleMock.EXPECT().
		NewReadContext(gomock.Any()).
		Return(readCtxMock, nil).
		AnyTimes()
	readCtxMock.EXPECT().
		Release().
		AnyTimes()
	persistenceModuleMock.EXPECT().
		GetBus().
		Return(busMock).
		AnyTimes()
	persistenceModuleMock.EXPECT().
		SetBus(busMock).
		AnyTimes()
	persistenceModuleMock.EXPECT().
		GetModuleName().
		Return(modules.PersistenceModuleName).
		AnyTimes()
	busMock.
		RegisterModule(persistenceModuleMock)

	return persistenceModuleMock
}

func validatorId(i int) string {
	return fmt.Sprintf(serviceURLFormat, i)
}

// createMockBus returns a mock bus with stubbed out functions for bus registration
func createMockBus(t *testing.T, runtimeMgr modules.RuntimeMgr) *mockModules.MockBus {
	t.Helper()
	ctrl := gomock.NewController(t)
	mockBus := mockModules.NewMockBus(ctrl)
	mockModulesRegistry := mockModules.NewMockModulesRegistry(ctrl)

	mockBus.EXPECT().
		GetRuntimeMgr().
		Return(runtimeMgr).
		AnyTimes()
	mockBus.EXPECT().
		RegisterModule(gomock.Any()).
		DoAndReturn(func(m modules.Submodule) {
			m.SetBus(mockBus)
		}).
		AnyTimes()
	mockModulesRegistry.EXPECT().
		GetModule(peerstore_provider.PeerstoreProviderSubmoduleName).
		Return(nil, runtime.ErrModuleNotRegistered(peerstore_provider.PeerstoreProviderSubmoduleName)).
		AnyTimes()
	mockModulesRegistry.EXPECT().
		GetModule(modules.CurrentHeightProviderSubmoduleName).
		Return(nil, runtime.ErrModuleNotRegistered(modules.CurrentHeightProviderSubmoduleName)).
		AnyTimes()
	mockBus.EXPECT().
		GetModulesRegistry().
		Return(mockModulesRegistry).
		AnyTimes()
	mockBus.EXPECT().
		PublishEventToBus(gomock.Any()).
		AnyTimes()

	return mockBus
}
