package step

import (
	"fmt"
	"testing"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/destination"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockPairManager is a hand-written mock for the private pairManager interface.
type mockPairManager struct {
	pairLists          []destination.PairList
	listCallCount      int
	createdID          string
	createErr          error
	createPairCalled   bool
	createWatchUDID    string
	createPhoneUDID    string
	activatePairCalled bool
	activateErr        error
}

func (m *mockPairManager) ListPairs() (destination.PairList, error) {
	idx := m.listCallCount
	m.listCallCount++
	if idx < len(m.pairLists) {
		return m.pairLists[idx], nil
	}
	return m.pairLists[len(m.pairLists)-1], nil
}

func (m *mockPairManager) CreatePair(watchUDID, phoneUDID string) (string, error) {
	m.createPairCalled = true
	m.createWatchUDID = watchUDID
	m.createPhoneUDID = phoneUDID
	if m.createErr != nil {
		return "", m.createErr
	}
	return m.createdID, nil
}

func (m *mockPairManager) ActivatePair(_ string) error {
	m.activatePairCalled = true
	return m.activateErr
}

// Test helpers

func pairList(id, phoneUDID, watchUDID, state string) destination.PairList {
	return destination.PairList{Pairs: map[string]destination.Pair{
		id: {
			Phone: destination.PairDevice{UDID: phoneUDID},
			Watch: destination.PairDevice{UDID: watchUDID},
			State: state,
		},
	}}
}

func emptyPairList() destination.PairList {
	return destination.PairList{Pairs: map[string]destination.Pair{}}
}

func defaultInputParser(t *testing.T) *MockInputParser {
	t.Helper()
	parser := NewMockInputParser(t)
	parser.EXPECT().Parse(mock.Anything).RunAndReturn(func(input interface{}) error {
		i := input.(*Input)
		i.IPhoneDevice = "iPhone 17 Pro"
		i.IOSVersion = "18.4"
		i.WatchDevice = "Apple Watch Series 11 (46mm)"
		i.WatchOS = "11.5"
		return nil
	})
	return parser
}

func defaultDeviceFinder(t *testing.T) *MockDeviceFinder {
	t.Helper()
	finder := NewMockDeviceFinder(t)
	finder.EXPECT().FindDevice(destination.Simulator{
		Platform: string(destination.IOSSimulator),
		Name:     "iPhone 17 Pro",
		OS:       "18.4",
	}).Return(destination.Device{UDID: "phone-uuid-1", Name: "iPhone 17 Pro"}, nil)
	finder.EXPECT().FindDevice(destination.Simulator{
		Platform: string(destination.WatchOSSimulator),
		Name:     "Apple Watch Series 11 (46mm)",
		OS:       "11.5",
	}).Return(destination.Device{UDID: "watch-uuid-1", Name: "Apple Watch Series 11 (46mm)"}, nil)
	return finder
}

func runAll(t *testing.T, s DevicePairerStep) error {
	t.Helper()
	config, err := s.ProcessConfig()
	if err != nil {
		return err
	}
	result, err := s.Run(config)
	if err != nil {
		return err
	}
	return s.ExportOutputs(result)
}

// Tests

func TestRun_PairSuccessfullyCreated(t *testing.T) {
	pairMgr := &mockPairManager{
		pairLists: []destination.PairList{
			emptyPairList(),
			pairList("new-pair-id", "phone-uuid-1", "watch-uuid-1", "(active, disconnected)"),
		},
		createdID: "new-pair-id",
	}
	envRepo := NewMockRepository(t)
	envRepo.EXPECT().Set("BITRISE_DEVICE_PAIR_UDID", "new-pair-id").Return(nil)
	envRepo.EXPECT().Set("BITRISE_IPHONE_UDID", "phone-uuid-1").Return(nil)
	envRepo.EXPECT().Set("BITRISE_WATCH_UDID", "watch-uuid-1").Return(nil)

	s := NewDevicePairerStep(defaultInputParser(t), log.NewLogger(), defaultDeviceFinder(t), pairMgr, envRepo)
	require.NoError(t, runAll(t, s))

	require.True(t, pairMgr.createPairCalled)
	require.Equal(t, "watch-uuid-1", pairMgr.createWatchUDID)
	require.Equal(t, "phone-uuid-1", pairMgr.createPhoneUDID)
}

func TestProcessConfig_OSNotFound(t *testing.T) {
	finder := NewMockDeviceFinder(t)
	finder.EXPECT().FindDevice(mock.Anything).Return(destination.Device{}, fmt.Errorf("no runtime installed for platform iOS"))

	s := NewDevicePairerStep(defaultInputParser(t), log.NewLogger(), finder, &mockPairManager{}, NewMockRepository(t))
	_, err := s.ProcessConfig()

	require.Error(t, err)
}

func TestRun_ExistingActivePair(t *testing.T) {
	pairMgr := &mockPairManager{
		pairLists: []destination.PairList{
			pairList("existing-pair-id", "phone-uuid-1", "watch-uuid-1", "(active, disconnected)"),
		},
	}
	envRepo := NewMockRepository(t)
	envRepo.EXPECT().Set("BITRISE_DEVICE_PAIR_UDID", "existing-pair-id").Return(nil)
	envRepo.EXPECT().Set("BITRISE_IPHONE_UDID", "phone-uuid-1").Return(nil)
	envRepo.EXPECT().Set("BITRISE_WATCH_UDID", "watch-uuid-1").Return(nil)

	s := NewDevicePairerStep(defaultInputParser(t), log.NewLogger(), defaultDeviceFinder(t), pairMgr, envRepo)
	require.NoError(t, runAll(t, s))

	require.False(t, pairMgr.createPairCalled)
}

func TestRun_InactivePairGetsActivated(t *testing.T) {
	pairMgr := &mockPairManager{
		pairLists: []destination.PairList{
			pairList("inactive-pair-id", "phone-uuid-1", "watch-uuid-1", "(inactive, disconnected)"),
		},
	}
	envRepo := NewMockRepository(t)
	envRepo.EXPECT().Set("BITRISE_DEVICE_PAIR_UDID", "inactive-pair-id").Return(nil)
	envRepo.EXPECT().Set("BITRISE_IPHONE_UDID", "phone-uuid-1").Return(nil)
	envRepo.EXPECT().Set("BITRISE_WATCH_UDID", "watch-uuid-1").Return(nil)

	s := NewDevicePairerStep(defaultInputParser(t), log.NewLogger(), defaultDeviceFinder(t), pairMgr, envRepo)
	require.NoError(t, runAll(t, s))

	require.True(t, pairMgr.activatePairCalled)
	require.False(t, pairMgr.createPairCalled)
}

func TestRun_MatchingPairFoundAmongMultiple(t *testing.T) {
	multiList := destination.PairList{Pairs: map[string]destination.Pair{
		"unrelated-pair-id": {
			Phone: destination.PairDevice{UDID: "other-phone"},
			Watch: destination.PairDevice{UDID: "other-watch"},
			State: "(active, disconnected)",
		},
		"target-pair-id": {
			Phone: destination.PairDevice{UDID: "phone-uuid-1"},
			Watch: destination.PairDevice{UDID: "watch-uuid-1"},
			State: "(active, disconnected)",
		},
	}}
	pairMgr := &mockPairManager{pairLists: []destination.PairList{multiList}}
	envRepo := NewMockRepository(t)
	envRepo.EXPECT().Set("BITRISE_DEVICE_PAIR_UDID", "target-pair-id").Return(nil)
	envRepo.EXPECT().Set("BITRISE_IPHONE_UDID", "phone-uuid-1").Return(nil)
	envRepo.EXPECT().Set("BITRISE_WATCH_UDID", "watch-uuid-1").Return(nil)

	s := NewDevicePairerStep(defaultInputParser(t), log.NewLogger(), defaultDeviceFinder(t), pairMgr, envRepo)
	require.NoError(t, runAll(t, s))

	require.False(t, pairMgr.createPairCalled)
}

func TestRun_UnavailablePairSkippedCreatesNew(t *testing.T) {
	pairMgr := &mockPairManager{
		pairLists: []destination.PairList{
			pairList("unavailable-pair-id", "phone-uuid-1", "watch-uuid-1", "(unavailable)"),
			pairList("new-pair-id", "phone-uuid-1", "watch-uuid-1", "(active, disconnected)"),
		},
		createdID: "new-pair-id",
	}
	envRepo := NewMockRepository(t)
	envRepo.EXPECT().Set("BITRISE_DEVICE_PAIR_UDID", "new-pair-id").Return(nil)
	envRepo.EXPECT().Set("BITRISE_IPHONE_UDID", "phone-uuid-1").Return(nil)
	envRepo.EXPECT().Set("BITRISE_WATCH_UDID", "watch-uuid-1").Return(nil)

	s := NewDevicePairerStep(defaultInputParser(t), log.NewLogger(), defaultDeviceFinder(t), pairMgr, envRepo)
	require.NoError(t, runAll(t, s))

	require.True(t, pairMgr.createPairCalled)
	require.False(t, pairMgr.activatePairCalled)
}
