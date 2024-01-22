# Ixian

ssh root@104.248.52.192

Visit [http://ixian.me:8081](http://ixian.me:8081). Past a NIP-19 npub key and enter. You'll get a list of your Long-Form articles. You'll be able to follow NIP-27 **Note References** in article content. For an example, use my npub14ge829c4pvgx24c35qts3sv82wc2xwcmgng93tzp6d52k9de2xgqq0y4jk.

## Stack

- Go
- [HTMX](https://htmx.org/)
- CSS

## Developers

1. Create a new profile configuration.

```shell
mkdir -p $HOME/.config/nostr
export CONFIG_NOSTR=$HOME/.config/nostr/alice.json
touch $CONFIG_NOSTR
```

2. Install the [Ixian CLI](https://github.com/dextyz/nix) tool to help you manage your profile from the terminal.

3. Before you can fetch notes you have to add at least one relay.

```shell
nix relay -add wss://relay.damus.io/
```

4. Run the server

```shell
make run
```

5. Navigate to [http://localhost:8081](http://localhost:8081)
