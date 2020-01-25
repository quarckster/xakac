# xakac
This CLI utility was insipired by [smee.io client](https://github.com/probot/smee-client). It forwards
payloads from remote servers to hosts in your network.

## Usage

`xakac -config /path/to/config.json`

In `config.json` you can define as many source-target pairs as you want:

```json
[
  {
    "source": "https://source_url_1",
    "target": "https://target_url_1"
  },
  {
    "source": "https://source_url_2",
    "target": "https://target_url_2"
  }
]
```

It's possible to specify sources and targets via environment variables:

`XAKAC_SOURCE_TARGET_1=https://source_url_1,https://target_url_1`

`XAKAC_SOURCE_TARGET_2=https://source_url_2,https://target_url_2`


`xakac` establishes connections to each source via [event source](https://developer.mozilla.org/en-US/docs/Web/API/EventSource)
and forwards payloads to a corresponding target.