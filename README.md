# ec_check

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`ec_check` is a tool to check Elastic Cloud deployments.

As of now, the main function is to evaluate, if downscaling of an Elasticsearch
data tier within an Elastic Cloud deployment is possible based on the current
disk usage.

## Usage

### Downscale Check

By default, for downscaling, only strict vertical downscaling is proposed.

By providing the extra flag `--recommend-zone-change`, `ec_check` will also
propose a combination of changing the instance size and the number of used zones
at the same time.  
**CAUTION**: This mode is not recommended by Elastic. If you strictly require 3
zones, this feature can not be used.

```bash
$ ec_check downscale --region <region> --profile <profile> --deployment <name> --username <username> --password <password>
```

By default, a headroom of 25% is required after downscaling for `ec_check` to
propose downscaling of a data tier. This can be changed by flag: `--headroom-pct`
and the respective percentage, e.g.: `--headroom-pct 27.5`

### Elasticsearch Regions

Get the supported list of regions:

```bash
$ ec_check regions
```

### Elasticsearch Profiles

Get the supported list of hardware profiles for a given region:

```bash
$ ec_check profiles --region <region>
```

## Community

This project has adopted the code of conduct defined by the [Contributor Covenant](https://contributor-covenant.org/)
to clarify expected behavior in our community. For more information, see the [Swiss Life OSS Code of Conduct](https://swisslife-oss.github.io/coc).

## Contributing

We welcome contributions from the community. Please open an issue or discussion with you idea/feature request and we will be happy to help you get started.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
