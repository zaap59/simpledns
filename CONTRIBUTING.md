# Guide de Contribution

## Commits Conventionnels

Ce projet utilise les [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) pour versionner automatiquement les releases avec [Release Please](https://github.com/googleapis/release-please).

### Format des commits

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types de commits

- **feat**: Une nouvelle fonctionnalité (génère une version mineure)
- **fix**: Correction d'un bug (génère une version patch)
- **docs**: Changements de documentation uniquement
- **style**: Changements qui n'affectent pas le code (formatage, etc.)
- **refactor**: Changement du code qui ne corrige pas de bug ou n'ajoute de fonctionnalité
- **perf**: Changement qui améliore les performances
- **test**: Ajout ou modification de tests
- **chore**: Changements de build, dépendances, etc.

### Exemples

**Nouvelle fonctionnalité :**
```
feat(zone): ajouter support des records CNAME

Permet la création d'alias DNS via les records CNAME
dans le format YAML.
```

**Correction de bug :**
```
fix(parser): gérer les espaces dans les valeurs YAML

Corrige un problème où les espaces au début ou à la fin
des valeurs étaient mal gérés dans les records DNS.
```

**Breaking change :**
```
feat!: remplacer format BIND par YAML

BREAKING CHANGE: Le format de configuration BIND n'est
plus supporté. Veuillez migrer vers le format YAML.
```

## Versioning

Ce projet suit [Semantic Versioning](https://semver.org/):
- **Version majeure** (X.0.0): Breaking changes
- **Version mineure** (0.X.0): Nouvelles fonctionnalités
- **Version patch** (0.0.X): Corrections de bugs

Les versions sont automatiquement créées et les releases GitHub sont générées automatiquement par Release Please basé sur les commits.

## Changelog

Voir [CHANGELOG.md](CHANGELOG.md) pour l'historique des changements.
