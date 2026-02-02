# Simple DNS server (Go)


Serveur DNS minimal en Go, supporte les enregistrements `A` et `CNAME`.

Prerequis:
- Go 1.20+
- le module `github.com/miekg/dns` (déclaré dans `go.mod`).


Configuration des zones:
- Le serveur peut charger des fichiers de zone BIND-style depuis un répertoire `conf/`.
- Alternativement il peut charger un fichier JSON (`zones.json`) map[string][]string.

Exécution (nécessite les privilèges root pour écouter sur le port 53):

```bash
cd /path/to/simpledns

**Exécution:**

- Écoute sur le port 53 par défaut (nécessite `root` / `sudo`).
- Flags disponibles (priorisent les valeurs du fichier de configuration si fournis):

```bash
# charge les fichiers BIND-style depuis le dossier `conf/` (ou défini en config)
sudo ./simpledns --confdir conf

# charge depuis un fichier JSON de zones
sudo ./simpledns --config zones.json

# définir les serveurs upstream en CLI (prioritaire sur la config)
sudo ./simpledns --forwarders 1.1.1.1,8.8.8.8
```

Configuration générale via `simpledns.json`:

Placez un fichier `simpledns.json` à la racine du projet pour définir les options globales. Exemple :

```json
{
  "confdir": "conf",
  "zones_file": "zones.json",
  "forwarders": ["1.1.1.1", "8.8.8.8"],
  "forward_timeout_seconds": 2
}
```

Champs supportés:
- `confdir`: dossier contenant les fichiers de zone BIND-style.
- `zones_file`: fichier JSON de zones (map[string][]string).
- `forwarders`: liste d'upstreams DNS (sans port ou `host:port`).
- `forward_timeout_seconds`: timeout en secondes pour les forwards.

Notes:
- Les flags CLI ont priorité sur les valeurs définies dans `simpledns.json`.
- Le champ `debug` a été supprimé (il n'existe plus dans les flags ni dans la config).
- Si un nom demandé n'existe pas localement, le serveur forwardera la requête vers les upstreams listés (si configurés).
	"debug": true
}
```

Les champs supportés:
- `confdir`: dossier contenant les fichiers de zone BIND-style.
- `zones_file`: fichier JSON de zones (map[string][]string).
- `forwarders`: liste d'upstreams DNS (sans port ou `host:port`).
- `forward_timeout_seconds`: timeout en secondes pour les forwards.
- `debug`: bool pour activer le logging détaillé.

-- `-forward` : spécifier une ou plusieurs adresses de serveur DNS upstream, séparées par des virgules.
	- Exemple: `-forward 1.1.1.1,8.8.8.8` ou avec ports `-forward 1.1.1.1:53,8.8.8.8:53`.
	- Option `-forward-timeout` permet de régler le timeout (par défaut 2s).

Le serveur tentera d'abord de répondre depuis les zones chargées; si le nom demandé n'existe pas localement, la requête sera renvoyée aux upstreams listés.


Pour développement sans `sudo`, changez le port dans `main.go` (par exemple `:8053`) puis rebuild/ lancez sans `sudo`:

```bash
go build -o simpledns .
./simpledns
dig @127.0.0.1 -p 8053 example.local A
```

**Format de configuration des zones:**
- Dossier `conf/`: fichiers de zone BIND-style (exemples fournis: `conf/homelab.int.zone`, `conf/lilcloud.net.zone`).
- Fichier JSON: map de nom de zone vers liste de chaînes RR compatibles avec `dns.NewRR`, exemple `zones.json` fourni.

**Tests rapides:**

```bash
dig @127.0.0.1 example.local A
dig @127.0.0.1 www.example.local CNAME
```

Fichiers d'exemple inclus dans le dépôt:
- `zones.json`
- `conf/homelab.int.zone`
- `conf/lilcloud.net.zone`

Si vous voulez, j'exécute `go build` ici pour vérifier la compilation ou je lance le serveur sur un port non privilégié pour des tests rapides.

