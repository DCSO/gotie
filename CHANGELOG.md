# Change Log

## v0.1.1 (2017-01-30)

**Implemented enhancements**

- Added JSON input channel delivering IOCs from JSON code parsed from an `io.Reader`
- Simplified option parsing.
- Addition of new query parameters to the `iocs` and `feeds` subcommands.
- JSON results are now aggregated in one IOC array across API pages.
- New `WriteIOCs()` and `WritePeriodFeeds()` APIfunctions accepting an `io.Reader`as output.
- Rewording of error messages.

**Bugs fixed**

- Link header parsing would faild due to bug in previously used library regarding commas in URLs. Fixed by switching to different implementation.

**Changes**

- IOCResult struct field `Error` is no longer a pointer to an error value.

## v0.1.0 (2016-11-29)

- Initial open source release
