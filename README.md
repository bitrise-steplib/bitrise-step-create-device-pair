# Create Simulator Device Pair

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/bitrise-step-create-device-pair?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/bitrise-step-create-device-pair/releases)

Creates an iPhone and Apple Watch simulator device pair

<details>
<summary>Description</summary>

Creates or finds an existing active simulator device pair between an iPhone and an Apple Watch.

The step looks up simulator devices by name and OS version, checks if an active pair already
exists between them, and creates one if needed.

</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/steps/adding-steps-to-a-workflow.html).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

### Create a device pair for testing

```yaml
- git::https://github.com/bitrise-steplib/bitrise-step-create-device-pair.git:
    inputs:
      - iphone_device: iPhone 17 Pro
      - ios_version: "18.4"
      - watch_device: Apple Watch Series 11 (46mm)
      - watchos_version: "11.5"
```

### Use the pair ID in a subsequent step

```yaml
- git::https://github.com/bitrise-steplib/bitrise-step-create-device-pair.git:
    inputs:
      - iphone_device: iPhone 17 Pro
      - ios_version: "18.4"
      - watch_device: Apple Watch Series 11 (46mm)
      - watchos_version: "11.5"
- xcode-test:
    inputs:
      - destination: platform=iOS Simulator,id=$BITRISE_IPHONE_UDID
```


## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `iphone_device` | The simulator device name as shown in `xcrun simctl list devices` and listed on bitrise.io/stacks.  | required |  |
| `ios_version` | The iOS runtime version. Use dot notation (e.g. "18.4", "17.5").  | required |  |
| `watch_device` | The simulator device name as shown in `xcrun simctl list devices` and listed on bitrise.io/stacks.  | required |  |
| `watchos_version` | The watchOS runtime version. Use dot notation (e.g. "11.5", "26.0").  | required |  |
| `delete_blocking_pairs` | When true and pairing fails because a device has reached its maximum number of allowed pairs, the step deletes the conflicting pair(s) and retries. When false, the step fails immediately on a capacity error.  | required | `true` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `BITRISE_DEVICE_PAIR_UDID` | The UUID identifier of the active device pair between the specified iPhone and Apple Watch simulators.  |
| `BITRISE_IPHONE_UDID` | The UDID of the iPhone simulator device that was matched by the given device name and iOS version.  |
| `BITRISE_WATCH_UDID` | The UDID of the Apple Watch simulator device that was matched by the given device name and watchOS version.  |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/bitrise-step-create-device-pair/pulls) and [issues](https://github.com/bitrise-steplib/bitrise-step-create-device-pair/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://docs.bitrise.io/en/bitrise-ci/bitrise-cli/running-your-first-local-build-with-the-cli.html).

Learn more about developing steps:

- [Create your own step](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/developing-your-own-bitrise-step/developing-a-new-step.html)
