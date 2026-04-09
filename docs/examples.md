### Create a device pair for testing

```yaml
- create-device-pair@1:
    inputs:
      - iphone_device: iPhone 17 Pro
      - ios_version: "18.4"
      - watch_device: Apple Watch Series 11 (46mm)
      - watchos_version: "11.5"
```

### Use the pair ID in a subsequent step

```yaml
- create-device-pair@1:
    inputs:
      - iphone_device: iPhone 17 Pro
      - ios_version: "18.4"
      - watch_device: Apple Watch Series 11 (46mm)
      - watchos_version: "11.5"
- xcode-test:
    inputs:
      - destination: platform=iOS Simulator,id=$BITRISE_IPHONE_UDID
```
