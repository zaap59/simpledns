# Format YAML pour zones DNS

## Structure

Le format YAML compact permet de gérer les configurations de zones DNS de manière lisible et structurée.

### Structure générale

```yaml
zone_config:
  name: example.com           # Nom de la zone
  origin: example.com.        # Origine DNS (avec point final)
  ttl: 3600                   # TTL par défaut en secondes

soa:
  ns: ns.example.com.         # Nameserver SOA
  admin: hostmaster@example.com  # Email admin (@ sera converti en .)
  serial: 1                   # Numéro de série SOA
  refresh: 3600               # Refresh SOA en secondes
  retry: 600                  # Retry SOA en secondes
  expire: 604800              # Expire SOA en secondes

dns_records:
  - name: ns                  # Nom du record (relatif à la zone)
    type: A                   # Type de record (A, AAAA, CNAME, MX, TXT, etc.)
    value: 192.168.1.2        # Valeur du record
    ttl: 3600                 # TTL optionnel (utilise zone_config.ttl par défaut)
  
  - name: www
    type: A
    value: 192.168.1.10
```

## Règles

1. **Noms relatifs** : Les noms dans `dns_records` sont relatifs à la zone
   - `@` ou omis = apex (root) de la zone
   - `www` = `www.example.com`
   - `api.subdomain` = `api.subdomain.example.com`

2. **Email admin** : Le caractère `@` est converti en `.` (format DNS standard)
   - `hostmaster@example.com` → `hostmaster.example.com.`

3. **TTL** : 
   - Utilise le TTL du record si spécifié
   - Sinon utilise `zone_config.ttl`

4. **Extension** : Les fichiers YAML doivent avoir l'extension `.yaml` ou `.yml`

## Exemples

### Zone A records simples

```yaml
zone_config:
  name: homelab.int
  origin: homelab.int.
  ttl: 3600

soa:
  ns: ns.homelab.int.
  admin: hostmaster@homelab.int
  serial: 1
  refresh: 3600
  retry: 600
  expire: 604800

dns_records:
  - name: ns
    type: A
    value: 192.168.1.2
  - name: router
    type: A
    value: 192.168.1.1
  - name: server
    type: A
    value: 192.168.1.10
```

### Zone avec multiples types de records

```yaml
zone_config:
  name: example.com
  origin: example.com.
  ttl: 3600

soa:
  ns: ns1.example.com.
  admin: admin@example.com
  serial: 2024020201
  refresh: 7200
  retry: 3600
  expire: 1209600

dns_records:
  # Apex records
  - name: "@"
    type: A
    value: 203.0.113.1
  
  # Subdomains
  - name: www
    type: A
    value: 203.0.113.10
  
  - name: api
    type: A
    value: 203.0.113.11
  
  # CNAME
  - name: mail
    type: CNAME
    value: example.com.
  
  # MX record
  - name: "@"
    type: MX
    value: "10 mail.example.com."
  
  # TXT records
  - name: "@"
    type: TXT
    value: "v=spf1 mx ~all"
  
  - name: _dmarc
    type: TXT
    value: "v=DMARC1; p=none"
```

## Format supporté

- Le serveur charge **uniquement** les fichiers au format YAML
- Les fichiers doivent avoir l'extension `.yaml` ou `.yml`
- Les fichiers avec d'autres extensions sont ignorés

