# Simple DNS server (Go)


Serveur DNS minimal en Go, supporte les enregistrements `A` et `CNAME`.

Prerequis:
- Go 1.20+
- le module `github.com/miekg/dns` (déclaré dans `go.mod`).


Configuration des zones:
- Le serveur charge les fichiers de zone au format YAML depuis le répertoire `conf/`.
- Les fichiers doivent avoir l'extension `.yaml` ou `.yml`.

Exécution (nécessite les privilèges root pour écouter sur le port 53):

```bash
cd /path/to/simpledns
```

**Exécution:**

- Écoute sur le port 53 par défaut (nécessite `root` / `sudo`).
- Flags disponibles (priorisent les valeurs du fichier de configuration si fournis):

```bash
# utilise le fichier de configuration par défaut (config.yaml)
sudo ./simpledns

# utilise un fichier de configuration personnalisé
sudo ./simpledns -config-file /path/to/custom.yaml

# charge les fichiers YAML depuis le dossier `conf/` (ou défini en config)
sudo ./simpledns -zones-dir conf

# définir les serveurs upstream en CLI (prioritaire sur la config)
sudo ./simpledns -forwarders 1.1.1.1,8.8.8.8

# activer les logs de debug
sudo ./simpledns -debug
```

Configuration générale via `config.yaml`:

Placez un fichier `config.yaml` à la racine du projet pour définir les options globales. Exemple :

```yaml
zones_dir: conf
forwarders:
  - 1.1.1.1
  - 8.8.8.8
forward_timeout_seconds: 2
```

Champs supportés:
- `zones_dir`: dossier contenant les fichiers de zone YAML.
- `forwarders`: liste d'upstreams DNS (sans port ou `host:port`).
- `forward_timeout_seconds`: timeout en secondes pour les forwards.

## Validation et tests

Le projet utilise des workflows GitHub Actions pour valider les changements :

- **Lint and Test** (`.github/workflows/lint-and-test.yml`)
  - Linting du code avec golangci-lint
  - Exécution des tests
  - Vérification que go.mod est à jour
  - Upload du coverage vers Codecov

- **Build Validation** (`.github/workflows/build-validation.yml`)
  - Tests de compilation multi-versions (Go 1.24, Go 1.25)
  - Build cross-platform (Linux, macOS, Windows × amd64, arm64)
  - Vérification de l'intégrité du build
  - Génération d'artefacts compilés

Voir [BUILD_VALIDATION.md](BUILD_VALIDATION.md) pour plus de détails.

Notes:
- Les flags CLI ont priorité sur les valeurs définies dans `config.yaml`.
- Le flag `-config-file` permet de spécifier un fichier de configuration personnalisé (par défaut: `config.yaml`).
- Si un nom demandé n'existe pas localement, le serveur forwardera la requête vers les upstreams listés (si configurés).


Pour développement sans `sudo`, changez le port dans `main.go` (par exemple `:8053`) puis rebuild/ lancez sans `sudo`:

```bash
go build -o simpledns .
./simpledns
dig @127.0.0.1 -p 8053 example.local A
```

**Format de configuration des zones:**
- Dossier `conf/`: fichiers de zone au format YAML (exemples fournis: `conf/homelab.int.yaml`, `conf/lilcloud.net.yaml`).
- Consultez [YAML_FORMAT.md](YAML_FORMAT.md) pour la documentation détaillée du format YAML.

**Tests rapides:**

```bash
dig @127.0.0.1 example.local A
dig @127.0.0.1 www.example.local CNAME
```

Fichiers d'exemple inclus dans le dépôt:
- `conf/homelab.int.yaml`
- `conf/lilcloud.net.yaml`

## Versioning

Ce projet utilise [Release Please](https://github.com/googleapis/release-please) pour gérer automatiquement les versions et les releases GitHub.

Les versions suivent [Semantic Versioning](https://semver.org/). Les commits doivent suivre la convention [Conventional Commits](https://www.conventionalcommits.org/) pour que les versions soient calculées correctement.

Voir [CONTRIBUTING.md](CONTRIBUTING.md) pour les détails sur le format des commits.

## Contribution

Pour contribuer au projet, veuillez consulter [CONTRIBUTING.md](CONTRIBUTING.md).

