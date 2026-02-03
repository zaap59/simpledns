# Guide Taskfile

Ce projet utilise [Taskfile](https://taskfile.dev/) pour gérer les tâches de développement de manière déclarative et lisible.

## Installation

### macOS
```bash
brew install go-task
```

### Linux
```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
```

### Ou télécharger
Voir [taskfile.dev/installation](https://taskfile.dev/installation/)

## Utilisation

### Voir toutes les tâches disponibles
```bash
task --list
```

### Lancer une tâche
```bash
task [nom_de_la_tache]
```

### Tâches principales

#### `task build`
Compile le serveur DNS. Génère le binaire `simpledns`.

#### `task lint`
Lance le linting avec `golangci-lint` sur tout le code.

#### `task fmt`
Formate le code avec `gofmt` et `goimports`.

#### `task test`
Lance tous les tests avec `-race` pour détecter les race conditions.

#### `task test-cover`
Lance les tests avec coverage et génère un rapport HTML (`coverage.html`).

#### `task all`
Exécute dans cet ordre :
1. `fmt` - Formate le code
2. `lint` - Linte le code
3. `test` - Lance les tests
4. `build` - Compile

C'est la tâche idéale à lancer avant de faire un commit.

#### `task run`
Compile et lance le serveur DNS (nécessite `sudo`).

#### `task run-debug`
Compile et lance le serveur en mode debug avec logs détaillés.

#### `task clean`
Supprime les binaires compilés et les fichiers de coverage.

#### `task deps`
Télécharge les dépendances Go.

#### `task tidy`
Nettoie `go.mod` et `go.sum` avec `go mod tidy`.

#### `task check`
Vérifie que `go.mod` et `go.sum` sont à jour (utilisé en CI/CD).

## Avantages par rapport au Makefile

- **YAML au lieu de Make** : Plus lisible et moins d'erreurs de tabulation
- **Dépendances entre tâches** : `task all` exécute les tâches dans l'ordre correct
- **Watch mode** : `task -w test` relance les tests à chaque modification
- **Parallélisation** : Les tâches indépendantes peuvent s'exécuter en parallèle
- **Variables globales** : Configuration centralisée et facile à partager

## Astuces

### Mode watch (relancer automatiquement)
```bash
task -w build    # Recompile à chaque modification
task -w test     # Relance les tests
```

### Ignorer les erreurs
```bash
task test || true
```

### Variables personnalisées
Vous pouvez définir des variables dans `Taskfile.yml` et les utiliser dans les tâches.

## Fichier de configuration

Voir `Taskfile.yml` à la racine du projet.
