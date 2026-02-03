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
# charge les fichiers YAML depuis le dossier `conf/` (ou défini en config)
sudo ./simpledns --confdir conf

# définir les serveurs upstream en CLI (prioritaire sur la config)
sudo ./simpledns --forwarders 1.1.1.1,8.8.8.8
```

Configuration générale via `simpledns.json`:

Placez un fichier `simpledns.json` à la racine du projet pour définir les options globales. Exemple :

```json
{
  "confdir": "conf",
  "forwarders": ["1.1.1.1", "8.8.8.8"],
  "forward_timeout_seconds": 2
}
```

Champs supportés:
- `confdir`: dossier contenant les fichiers de zone YAML.
- `forwarders`: liste d'upstreams DNS (sans port ou `host:port`).
- `forward_timeout_seconds`: timeout en secondes pour les forwards.

## Développement

### Build et test avec Taskfile

Ce projet utilise [Taskfile](https://taskfile.dev/) pour gérer les tâches de développement.

**Installation de Task :**
```bash
# macOS
brew install go-task

# Ou télécharger depuis https://taskfile.dev/installation/
```

**Commandes disponibles :**
```bash
# Compiler
task build

# Linter le code
task lint

# Formater le code
task fmt

# Lancer les tests
task test

# Lancer les tests avec coverage
task test-cover

# Tout faire (format, lint, test, build)
task all

# Lancer le serveur (nécessite sudo)
task run

# Lancer le serveur en mode debug
task run-debug

# Voir toutes les commandes disponibles
task --list
```

### Commandes Go directes (alternative)

Si vous préférez ne pas utiliser Task :

```bash
# Compiler
go build -o simpledns main.go

# Tester
go test -v ./...

# Tester avec coverage
go test -v -race -coverprofile=coverage.out ./...

# Télécharger les dépendances
go mod download

# Nettoyer les dépendances
go mod tidy
```

### Validation automatique

Ce projet utilise GitHub Actions pour :
- **Linter** le code avec `golangci-lint`
- **Compiler** le code
- **Lancer les tests**
- **Vérifier** que `go.mod` est à jour
- **Générer les releases** avec Release Please (Semantic Versioning)

Notes:
- Les flags CLI ont priorité sur les valeurs définies dans `simpledns.json`.
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

