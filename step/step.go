package step

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/destination"
)

// Input holds the raw step inputs parsed from environment variables.
type Input struct {
	IPhoneDevice        string `env:"iphone_device,required"`
	IOSVersion          string `env:"ios_version,required"`
	WatchDevice         string `env:"watch_device,required"`
	WatchOS             string `env:"watchos_version,required"`
	DeleteBlockingPairs bool   `env:"delete_blocking_pairs"`
}

// PairPlan holds resolved device UDIDs and run options after input parsing.
type PairPlan struct {
	PhoneUDID           string
	WatchUDID           string
	DeleteBlockingPairs bool
}

// Result holds the outcome of the step run.
type Result struct {
	PairUDID  string
	PhoneUDID string
	WatchUDID string
}

type pairManager interface {
	ListPairs() (destination.PairList, error)
	CreatePair(watchUDID, phoneUDID string) (string, error)
	ActivatePair(pairUDID string) error
	Unpair(pairUDID string) error
}

// DevicePairerStep is the main step struct with injected dependencies.
type DevicePairerStep struct {
	inputParser   stepconf.InputParser
	logger        log.Logger
	deviceFinder  destination.DeviceFinder
	pairManager   pairManager
	envRepository env.Repository
}

// NewDevicePairerStep creates a new DevicePairerStep with its dependencies.
func NewDevicePairerStep(
	inputParser stepconf.InputParser,
	logger log.Logger,
	deviceFinder destination.DeviceFinder,
	pairManager pairManager,
	envRepository env.Repository,
) DevicePairerStep {
	return DevicePairerStep{
		inputParser:   inputParser,
		logger:        logger,
		deviceFinder:  deviceFinder,
		pairManager:   pairManager,
		envRepository: envRepository,
	}
}

// ProcessConfig parses step inputs and resolves simulator device UDIDs.
func (s DevicePairerStep) ProcessConfig() (PairPlan, error) {
	var input Input
	if err := s.inputParser.Parse(&input); err != nil {
		return PairPlan{}, fmt.Errorf("parse inputs: %w", err)
	}
	stepconf.Print(input)

	phoneUDID, err := s.findDevice(input.IPhoneDevice, input.IOSVersion, destination.IOSSimulator)
	if err != nil {
		return PairPlan{}, fmt.Errorf("find iPhone simulator: %w", err)
	}

	watchUDID, err := s.findDevice(input.WatchDevice, input.WatchOS, destination.WatchOSSimulator)
	if err != nil {
		return PairPlan{}, fmt.Errorf("find Watch simulator: %w", err)
	}

	return PairPlan{
		PhoneUDID:           phoneUDID,
		WatchUDID:           watchUDID,
		DeleteBlockingPairs: input.DeleteBlockingPairs,
	}, nil
}

// Run finds or creates an active simulator device pair.
func (s DevicePairerStep) Run(config PairPlan) (Result, error) {
	pairUDID, inactive, err := s.findExistingPair(config.PhoneUDID, config.WatchUDID)
	if err != nil {
		return Result{}, err
	}

	switch {
	case pairUDID != "" && inactive:
		if err := s.activatePair(pairUDID); err != nil {
			return Result{}, err
		}
	case pairUDID != "":
		// already active, nothing to do
	default:
		pairUDID, err = s.createPair(config.PhoneUDID, config.WatchUDID, config.DeleteBlockingPairs)
		if err != nil {
			return Result{}, err
		}
	}

	return Result{
		PairUDID:  pairUDID,
		PhoneUDID: config.PhoneUDID,
		WatchUDID: config.WatchUDID,
	}, nil
}

// ExportOutputs exports the step result as Bitrise environment variables.
func (s DevicePairerStep) ExportOutputs(result Result) error {
	s.logger.Println()
	s.logger.Infof("Exporting outputs:")

	outputs := []struct{ key, value string }{
		{"BITRISE_DEVICE_PAIR_UDID", result.PairUDID},
		{"BITRISE_IPHONE_UDID", result.PhoneUDID},
		{"BITRISE_WATCH_UDID", result.WatchUDID},
	}

	for _, o := range outputs {
		if err := s.envRepository.Set(o.key, o.value); err != nil {
			return fmt.Errorf("export %s: %w", o.key, err)
		}
		s.logger.Donef("%s=%s", o.key, o.value)
	}

	return nil
}

func (s DevicePairerStep) findDevice(name, osVersion string, platform destination.Platform) (string, error) {
	device, err := s.deviceFinder.FindDevice(destination.Simulator{
		Platform: string(platform),
		Name:     name,
		OS:       osVersion,
	})
	if err != nil {
		return "", err
	}
	s.logger.Donef("Resolved %s (%s): %s", name, osVersion, device.UDID)
	return device.UDID, nil
}

func (s DevicePairerStep) findExistingPair(phoneUDID, watchUDID string) (pairUDID string, inactive bool, err error) {
	s.logger.Println()
	s.logger.Infof("Checking for existing device pair...")

	pairList, err := s.pairManager.ListPairs()
	if err != nil {
		return "", false, fmt.Errorf("list device pairs: %w", err)
	}

	for id, pair := range pairList.Pairs {
		if pair.Phone.UDID != phoneUDID || pair.Watch.UDID != watchUDID {
			continue
		}
		if pair.IsUnavailable() {
			s.logger.Warnf("Skipping pair %s: unavailable (runtime not installed)", id)
			continue
		}
		s.logger.Donef("Found existing pair: %s (state: %s)", id, pair.State)
		return id, pair.IsInactive(), nil
	}

	s.logger.Printf("No existing pair found")

	return "", false, nil
}

func (s DevicePairerStep) activatePair(pairUDID string) error {
	s.logger.Printf("Activating existing pair %s...", pairUDID)

	if err := s.pairManager.ActivatePair(pairUDID); err != nil {
		return fmt.Errorf("activate pair %s: %w", pairUDID, err)
	}

	s.logger.Donef("Activated pair: %s", pairUDID)

	return nil
}

func (s DevicePairerStep) createPair(phoneUDID, watchUDID string, deleteBlocking bool) (string, error) {
	s.logger.Printf("Creating a new device pair...")

	pairUDID, err := s.pairManager.CreatePair(watchUDID, phoneUDID)
	if err != nil {
		if !deleteBlocking || !strings.Contains(err.Error(), "maximum number") {
			return "", fmt.Errorf("create device pair: %w", err)
		}

		s.logger.Warnf("Pairing failed due to capacity limit, clearing blocking pairs...")
		if deleteErr := s.deleteBlockingPairs(phoneUDID, watchUDID); deleteErr != nil {
			return "", fmt.Errorf("create device pair: %w (failed to clear blocking pairs: %s)", err, deleteErr)
		}

		pairUDID, err = s.pairManager.CreatePair(watchUDID, phoneUDID)
		if err != nil {
			return "", fmt.Errorf("create device pair after clearing blocking pairs: %w", err)
		}
	}

	s.logger.Println()
	s.logger.Infof("Verifying new pair...")

	pairList, err := s.pairManager.ListPairs()
	if err != nil {
		return "", fmt.Errorf("verify pair: %w", err)
	}

	pair, ok := pairList.Pairs[pairUDID]
	if !ok {
		return "", fmt.Errorf("newly created pair %s not found in pair list", pairUDID)
	}
	if pair.IsInactive() {
		return "", fmt.Errorf("newly created pair %s did not reach active state (state: %s)", pairUDID, pair.State)
	}

	s.logger.Donef("Pair created and active: %s", pairUDID)

	return pairUDID, nil
}

func (s DevicePairerStep) deleteBlockingPairs(phoneUDID, watchUDID string) error {
	pairList, err := s.pairManager.ListPairs()
	if err != nil {
		return fmt.Errorf("list blocking pairs: %w", err)
	}

	for id, pair := range pairList.Pairs {
		if pair.Phone.UDID != phoneUDID && pair.Watch.UDID != watchUDID {
			continue
		}
		s.logger.Warnf("Deleting blocking pair %s (phone: %s, watch: %s)", id, pair.Phone.UDID, pair.Watch.UDID)
		if err := s.pairManager.Unpair(id); err != nil {
			return fmt.Errorf("delete pair %s: %w", id, err)
		}
	}

	return nil
}
