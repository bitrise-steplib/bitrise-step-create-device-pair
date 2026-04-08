package main

import (
	"os"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-steputils/v2/stepenv"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/destination"
	"github.com/bitrise-io/go-xcode/v2/xcodeversion"

	"github.com/bitrise-steplib/bitrise-step-create-device-pair/step"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := log.NewLogger()
	devicePairerStep, err := createStep(logger)
	if err != nil {
		logger.Errorf(err.Error())
		return 1
	}

	config, err := devicePairerStep.ProcessConfig()
	if err != nil {
		logger.Errorf(err.Error())
		return 1
	}

	result, err := devicePairerStep.Run(config)
	if err != nil {
		logger.Errorf(err.Error())
		return 1
	}

	if err := devicePairerStep.ExportOutputs(result); err != nil {
		logger.Errorf(err.Error())
		return 1
	}

	return 0
}

func createStep(logger log.Logger) (step.DevicePairerStep, error) {
	envRepository := env.NewRepository()
	inputParser := stepconf.NewInputParser(envRepository)
	commandFactory := command.NewFactory(envRepository)

	xcodeVersion, err := xcodeversion.NewXcodeVersionProvider(commandFactory).GetVersion()
	if err != nil {
		return step.DevicePairerStep{}, err
	}

	deviceFinder := destination.NewDeviceFinder(logger, commandFactory, xcodeVersion)
	pairManager := destination.NewPairManager(commandFactory)
	stepenvRepo := stepenv.NewRepository(envRepository)

	return step.NewDevicePairerStep(inputParser, logger, deviceFinder, pairManager, stepenvRepo), nil
}
