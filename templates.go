package main

// HTML templates for web interface

// Index page - list of zones (dashboard)
const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SimpleDNS - Dashboard</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        primary: '#3498db',
                        'primary-dark': '#2980b9',
                    }
                }
            }
        }
    </script>
</head>
<body class="bg-gray-100 text-gray-800 font-sans">
    <div class="max-w-6xl mx-auto p-5">
        <!-- Header -->
        <header class="bg-slate-800 text-white p-5 mb-5 rounded-lg">
            <div class="flex justify-between items-start">
                <div>
                    <h1 class="text-2xl font-bold">🌐 SimpleDNS</h1>
                    <div class="flex gap-4 mt-3">
                        <span class="bg-white/10 px-4 py-2 rounded">📁 Zones: {{.ZoneCount}}</span>
                        <span class="bg-white/10 px-4 py-2 rounded">📝 Records: {{.RecordCount}}</span>
                        <span class="bg-white/10 px-4 py-2 rounded">⚙️ Mode: {{.Mode}}</span>
                    </div>
                </div>
                {{if .EditMode}}
                <button onclick="showAddZoneModal()" class="bg-green-500 hover:bg-green-600 px-4 py-2 rounded font-medium transition-colors">+ Add Zone</button>
                {{end}}
            </div>
        </header>
        
        {{if .EditMode}}
        <!-- Add Zone Modal -->
        <div id="addZoneModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
            <div class="bg-white rounded-lg p-6 w-full max-w-md mx-4">
                <h2 class="text-xl font-bold mb-4">Add New Zone</h2>
                <form id="addZoneForm" onsubmit="submitZone(event)">
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">Zone Name *</label>
                        <input type="text" name="name" required placeholder="example.com" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                    </div>
                    <div class="flex gap-2 justify-end">
                        <button type="button" onclick="hideAddZoneModal()" class="px-4 py-2 border rounded hover:bg-gray-100">Cancel</button>
                        <button type="submit" class="px-4 py-2 bg-primary text-white rounded hover:bg-primary-dark">Create Zone</button>
                    </div>
                </form>
            </div>
        </div>
        {{end}}

        <!-- Zones List -->
        <div class="mb-5">
            <h2 class="text-xl font-bold mb-4 text-gray-700">DNS Zones</h2>
            {{if .Zones}}
            <div class="grid gap-4">
                {{range .Zones}}
                <div class="bg-white rounded-lg shadow hover:shadow-md transition-shadow">
                    <div class="p-5 flex justify-between items-center">
                        <div class="flex items-center gap-4">
                            {{if .Enabled}}
                            <div class="text-3xl cursor-pointer" onclick="toggleZone({{.ID}}, this)" title="Zone active - Cliquez pour désactiver">✅</div>
                            {{else}}
                            <div class="text-3xl cursor-pointer" onclick="toggleZone({{.ID}}, this)" title="Zone inactive - Cliquez pour activer">⛔</div>
                            {{end}}
                            <div>
                                <h3 class="text-lg font-bold text-gray-800">{{.Name}}</h3>
                                <p class="text-sm text-gray-500">{{len .Records}} records {{if not .Enabled}}<span class="text-red-500 font-medium">• Désactivée</span>{{end}}</p>
                            </div>
                        </div>
                        <div class="flex gap-2">
                            <a href="/zones/{{.Name}}/records" class="bg-primary hover:bg-primary-dark text-white px-4 py-2 rounded text-sm transition-colors">📝 Records</a>
                            <a href="/zones/{{.Name}}/settings" class="bg-gray-500 hover:bg-gray-600 text-white px-4 py-2 rounded text-sm transition-colors">⚙️ Settings</a>
                        </div>
                    </div>
                </div>
                {{end}}
            </div>
            {{else}}
            <div class="bg-white rounded-lg shadow p-10 text-center text-gray-400">
                No zones configured. {{if .EditMode}}Click "Add Zone" to create one.{{end}}
            </div>
            {{end}}
        </div>

        <!-- Forwarders Section -->
        <div class="bg-white rounded-lg shadow mb-5">
            <div class="flex justify-between items-center px-5 py-4 border-b">
                <h2 class="text-lg font-bold text-gray-700">🔀 Forwarders ({{len .Forwarders}})</h2>
                {{if .EditMode}}
                <button onclick="showAddForwarderModal()" class="bg-green-500 hover:bg-green-600 text-white px-3 py-1 rounded text-sm">+ Add</button>
                {{end}}
            </div>
            <div class="p-5">
                {{if .Forwarders}}
                <div class="flex flex-wrap gap-2" id="forwarders-list">
                    {{range .Forwarders}}
                    <div class="flex items-center gap-2 bg-gray-100 px-3 py-2 rounded" data-forwarder="{{.}}">
                        <span class="font-mono text-sm">{{.}}</span>
                        {{if $.EditMode}}
                        <button onclick="deleteForwarder('{{.}}', this)" class="text-red-500 hover:text-red-700 text-sm">✕</button>
                        {{end}}
                    </div>
                    {{end}}
                </div>
                {{else}}
                <div class="text-center text-gray-400" id="no-forwarders">No forwarders configured</div>
                {{end}}
            </div>
        </div>
        
        {{if .EditMode}}
        <!-- Add Forwarder Modal -->
        <div id="addForwarderModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
            <div class="bg-white rounded-lg p-6 w-full max-w-md mx-4">
                <h2 class="text-xl font-bold mb-4">Add Forwarder</h2>
                <form id="addForwarderForm" onsubmit="submitForwarder(event)">
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">DNS Server Address *</label>
                        <input type="text" name="address" required placeholder="8.8.8.8 or 8.8.8.8:53" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                        <p class="text-xs text-gray-500 mt-1">IP address or hostname, optionally with port (default: 53)</p>
                    </div>
                    <div class="flex gap-2 justify-end">
                        <button type="button" onclick="hideAddForwarderModal()" class="px-4 py-2 border rounded hover:bg-gray-100">Cancel</button>
                        <button type="submit" class="px-4 py-2 bg-primary text-white rounded hover:bg-primary-dark">Add Forwarder</button>
                    </div>
                </form>
            </div>
        </div>
        {{end}}

        <!-- Footer -->
        <footer class="text-center py-5 text-gray-400 text-sm">
            SimpleDNS &bull; <a href="/api/zones" class="text-primary hover:underline">API</a> &bull; <a href="/api/health" class="text-primary hover:underline">Health</a>
        </footer>
    </div>
    
    <script>
        // Modal functions
        function showAddZoneModal() {
            document.getElementById('addZoneModal').classList.remove('hidden');
            document.getElementById('addZoneModal').classList.add('flex');
        }
        function hideAddZoneModal() {
            document.getElementById('addZoneModal').classList.add('hidden');
            document.getElementById('addZoneModal').classList.remove('flex');
            document.getElementById('addZoneForm').reset();
        }
        
        async function submitZone(event) {
            event.preventDefault();
            const form = event.target;
            const data = { name: form.name.value };
            try {
                const resp = await fetch('/api/zones', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(data)
                });
                if (resp.ok) {
                    window.location.reload();
                } else {
                    const err = await resp.json();
                    alert('Failed to create zone: ' + (err.error || 'Unknown error'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        function showAddForwarderModal() {
            document.getElementById('addForwarderModal').classList.remove('hidden');
            document.getElementById('addForwarderModal').classList.add('flex');
        }
        
        function hideAddForwarderModal() {
            document.getElementById('addForwarderModal').classList.add('hidden');
            document.getElementById('addForwarderModal').classList.remove('flex');
            document.getElementById('addForwarderForm').reset();
        }
        
        async function submitForwarder(event) {
            event.preventDefault();
            const form = event.target;
            let address = form.address.value.trim();
            if (!address.includes(':')) {
                address = address + ':53';
            }
            try {
                const resp = await fetch('/api/forwarders', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({ address: address })
                });
                if (resp.ok) {
                    window.location.reload();
                } else {
                    const err = await resp.json();
                    alert('Failed to add forwarder: ' + (err.error || 'Unknown error'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        async function deleteForwarder(address, btn) {
            if (!confirm('Remove forwarder ' + address + '?')) return;
            try {
                const resp = await fetch('/api/forwarders/' + encodeURIComponent(address), { method: 'DELETE' });
                if (resp.ok) {
                    btn.closest('[data-forwarder]').remove();
                } else {
                    alert('Failed to remove forwarder');
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        async function toggleZone(id, el) {
            try {
                const resp = await fetch('/api/zones/' + id + '/toggle', { method: 'PATCH' });
                if (resp.ok) {
                    const data = await resp.json();
                    if (data.enabled) {
                        el.textContent = '✅';
                        el.title = 'Zone active - Cliquez pour désactiver';
                        el.closest('.flex').querySelector('.text-red-500')?.remove();
                    } else {
                        el.textContent = '⛔';
                        el.title = 'Zone inactive - Cliquez pour activer';
                        const sub = el.closest('.flex').querySelector('.text-gray-500');
                        if (sub && !sub.querySelector('.text-red-500')) {
                            sub.innerHTML = sub.innerHTML + ' <span class="text-red-500 font-medium">• Désactivée</span>';
                        }
                    }
                } else {
                    alert('Failed to toggle zone');
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
    </script>
</body>
</html>
`

// Zone Records page
const zoneRecordsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SimpleDNS - {{.Zone.Name}} Records</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        primary: '#3498db',
                        'primary-dark': '#2980b9',
                    }
                }
            }
        }
    </script>
</head>
<body class="bg-gray-100 text-gray-800 font-sans">
    <div class="max-w-6xl mx-auto p-5">
        <!-- Header -->
        <header class="bg-slate-800 text-white p-5 mb-5 rounded-lg">
            <div class="flex justify-between items-start">
                <div>
                    <div class="flex items-center gap-2 mb-2">
                        <a href="/" class="text-white/70 hover:text-white">🏠 Home</a>
                        <span class="text-white/50">/</span>
                        <span>{{.Zone.Name}}</span>
                    </div>
                    <h1 class="text-2xl font-bold">📝 DNS Records</h1>
                    <div class="flex gap-4 mt-3">
                        <span class="bg-white/10 px-4 py-2 rounded">Zone: {{.Zone.Name}}</span>
                        <span class="bg-white/10 px-4 py-2 rounded">{{len .Zone.Records}} records</span>
                    </div>
                </div>
                <div class="flex gap-2">
                    {{if .EditMode}}
                    <button onclick="showAddRecordModal()" class="bg-green-500 hover:bg-green-600 px-4 py-2 rounded font-medium transition-colors">+ Add Record</button>
                    {{end}}
                    <a href="/zones/{{.Zone.Name}}/settings" class="bg-gray-600 hover:bg-gray-700 px-4 py-2 rounded font-medium transition-colors">⚙️ Settings</a>
                </div>
            </div>
        </header>

        <!-- Filter buttons -->
        <div class="flex gap-2 mb-4 flex-wrap">
            <button class="filter-btn px-3 py-1 border rounded text-sm bg-primary text-white border-primary" data-filter="all">All</button>
            <button class="filter-btn px-3 py-1 border rounded text-sm bg-white border-gray-300" data-filter="A">A</button>
            <button class="filter-btn px-3 py-1 border rounded text-sm bg-white border-gray-300" data-filter="AAAA">AAAA</button>
            <button class="filter-btn px-3 py-1 border rounded text-sm bg-white border-gray-300" data-filter="CNAME">CNAME</button>
            <button class="filter-btn px-3 py-1 border rounded text-sm bg-white border-gray-300" data-filter="MX">MX</button>
            <button class="filter-btn px-3 py-1 border rounded text-sm bg-white border-gray-300" data-filter="TXT">TXT</button>
            <button class="filter-btn px-3 py-1 border rounded text-sm bg-white border-gray-300" data-filter="NS">NS</button>
        </div>

        {{if .EditMode}}
        <!-- Add Record Modal -->
        <div id="addRecordModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
            <div class="bg-white rounded-lg p-6 w-full max-w-md mx-4">
                <h2 class="text-xl font-bold mb-4">Add Record to {{.Zone.Name}}</h2>
                <form id="addRecordForm" onsubmit="submitRecord(event)">
                    <input type="hidden" name="zone_id" value="{{.Zone.ID}}">
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">Name *</label>
                        <input type="text" name="name" required placeholder="www or @" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                    </div>
                    <div class="grid grid-cols-2 gap-4 mb-4">
                        <div>
                            <label class="block text-sm font-medium mb-1">Type *</label>
                            <select name="type" required class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                                <option value="A">A</option>
                                <option value="AAAA">AAAA</option>
                                <option value="CNAME">CNAME</option>
                                <option value="MX">MX</option>
                                <option value="TXT">TXT</option>
                                <option value="NS">NS</option>
                                <option value="PTR">PTR</option>
                            </select>
                        </div>
                        <div>
                            <label class="block text-sm font-medium mb-1">TTL</label>
                            <input type="number" name="ttl" value="3600" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                        </div>
                    </div>
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">Value *</label>
                        <input type="text" name="value" required placeholder="192.168.1.1" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                    </div>
                    <div class="flex gap-2 justify-end">
                        <button type="button" onclick="hideAddRecordModal()" class="px-4 py-2 border rounded hover:bg-gray-100">Cancel</button>
                        <button type="submit" class="px-4 py-2 bg-primary text-white rounded hover:bg-primary-dark">Add Record</button>
                    </div>
                </form>
            </div>
        </div>
        
        <!-- Edit Record Modal -->
        <div id="editRecordModal" class="fixed inset-0 bg-black/50 hidden items-center justify-center z-50">
            <div class="bg-white rounded-lg p-6 w-full max-w-md mx-4">
                <h2 class="text-xl font-bold mb-4">Edit Record</h2>
                <form id="editRecordForm" onsubmit="submitEditRecord(event)">
                    <input type="hidden" name="record_id" id="editRecordId">
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">Name *</label>
                        <input type="text" name="name" id="editRecordName" required placeholder="www or @" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                    </div>
                    <div class="grid grid-cols-2 gap-4 mb-4">
                        <div>
                            <label class="block text-sm font-medium mb-1">Type *</label>
                            <select name="type" id="editRecordType" required class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                                <option value="A">A</option>
                                <option value="AAAA">AAAA</option>
                                <option value="CNAME">CNAME</option>
                                <option value="MX">MX</option>
                                <option value="TXT">TXT</option>
                                <option value="NS">NS</option>
                            </select>
                        </div>
                        <div>
                            <label class="block text-sm font-medium mb-1">TTL</label>
                            <input type="number" name="ttl" id="editRecordTTL" value="3600" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                        </div>
                    </div>
                    <div class="mb-4">
                        <label class="block text-sm font-medium mb-1">Value *</label>
                        <input type="text" name="value" id="editRecordValue" required placeholder="192.168.1.1" class="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-primary">
                    </div>
                    <div class="flex gap-2 justify-end">
                        <button type="button" onclick="hideEditRecordModal()" class="px-4 py-2 border rounded hover:bg-gray-100">Cancel</button>
                        <button type="submit" class="px-4 py-2 bg-primary text-white rounded hover:bg-primary-dark">Save Changes</button>
                    </div>
                </form>
            </div>
        </div>
        {{end}}

        <!-- Records Table -->
        <div class="bg-white rounded-lg shadow overflow-hidden">
            {{if .Zone.Records}}
            <table class="w-full">
                <thead class="bg-gray-50">
                    <tr>
                        <th class="px-5 py-3 text-left font-semibold text-gray-600 border-b">Name</th>
                        <th class="px-5 py-3 text-left font-semibold text-gray-600 border-b">Type</th>
                        <th class="px-5 py-3 text-left font-semibold text-gray-600 border-b">Value</th>
                        <th class="px-5 py-3 text-left font-semibold text-gray-600 border-b">TTL</th>
                        {{if $.EditMode}}<th class="px-5 py-3 text-left font-semibold text-gray-600 border-b w-24">Actions</th>{{end}}
                    </tr>
                </thead>
                <tbody id="records-tbody">
                    {{range .Zone.Records}}
                    <tr data-type="{{.Type}}" data-record-id="{{.ID}}" class="hover:bg-gray-50 border-b border-gray-100">
                        <td class="px-5 py-3 font-mono text-sm" data-field="name">{{.Name}}</td>
                        <td class="px-5 py-3" data-field="type">
                            {{if eq .Type "A"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-blue-100 text-blue-700">{{.Type}}</span>
                            {{else if eq .Type "AAAA"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-green-100 text-green-700">{{.Type}}</span>
                            {{else if eq .Type "CNAME"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-orange-100 text-orange-700">{{.Type}}</span>
                            {{else if eq .Type "MX"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-pink-100 text-pink-700">{{.Type}}</span>
                            {{else if eq .Type "TXT"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-purple-100 text-purple-700">{{.Type}}</span>
                            {{else if eq .Type "NS"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-teal-100 text-teal-700">{{.Type}}</span>
                            {{else if eq .Type "SOA"}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-gray-200 text-gray-700">{{.Type}}</span>
                            {{else}}<span class="px-2 py-0.5 rounded text-sm font-medium bg-gray-100 text-gray-600">{{.Type}}</span>
                            {{end}}
                        </td>
                        <td class="px-5 py-3 font-mono text-sm text-gray-600" data-field="value">{{.Value}}</td>
                        <td class="px-5 py-3 text-gray-500" data-field="ttl">{{.TTL}}</td>
                        {{if $.EditMode}}<td class="px-5 py-3">
                            <div class="flex gap-1">
                                <button onclick="showEditRecordModal({{.ID}}, this)" class="text-blue-500 hover:text-blue-700" title="Edit">✏️</button>
                                <button onclick="deleteRecord({{.ID}}, this)" class="text-red-500 hover:text-red-700" title="Delete">🗑</button>
                            </div>
                        </td>{{end}}
                    </tr>
                    {{end}}
                </tbody>
            </table>
            {{else}}
            <div class="text-center py-10 text-gray-400">No records in this zone. {{if .EditMode}}Click "Add Record" to create one.{{end}}</div>
            {{end}}
        </div>

        <!-- Footer -->
        <footer class="text-center py-5 text-gray-400 text-sm">
            <a href="/" class="text-primary hover:underline">← Back to Dashboard</a>
        </footer>
    </div>
    
    <script>
        const zoneId = {{.Zone.ID}};
        
        // Filter functionality
        let activeFilter = 'all';
        document.querySelectorAll('.filter-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.filter-btn').forEach(b => {
                    b.classList.remove('bg-primary', 'text-white', 'border-primary');
                    b.classList.add('bg-white', 'border-gray-300');
                });
                btn.classList.remove('bg-white', 'border-gray-300');
                btn.classList.add('bg-primary', 'text-white', 'border-primary');
                activeFilter = btn.dataset.filter;
                applyFilter();
            });
        });
        
        function applyFilter() {
            document.querySelectorAll('tr[data-type]').forEach(row => {
                if (activeFilter === 'all' || row.dataset.type === activeFilter) {
                    row.classList.remove('hidden');
                } else {
                    row.classList.add('hidden');
                }
            });
        }
        
        // Modal functions
        function showAddRecordModal() {
            document.getElementById('addRecordModal').classList.remove('hidden');
            document.getElementById('addRecordModal').classList.add('flex');
        }
        function hideAddRecordModal() {
            document.getElementById('addRecordModal').classList.add('hidden');
            document.getElementById('addRecordModal').classList.remove('flex');
            document.getElementById('addRecordForm').reset();
        }
        
        async function submitRecord(event) {
            event.preventDefault();
            const form = event.target;
            const data = {
                zone_id: zoneId,
                name: form.name.value,
                type: form.type.value,
                value: form.value.value,
                ttl: parseInt(form.ttl.value) || 3600
            };
            try {
                const resp = await fetch('/api/zones/' + zoneId + '/records', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(data)
                });
                if (resp.ok) {
                    const record = await resp.json();
                    addRecordRow(record);
                    hideAddRecordModal();
                } else {
                    const err = await resp.json();
                    alert('Failed to add record: ' + (err.error || 'Unknown error'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        function addRecordRow(record) {
            const tbody = document.getElementById('records-tbody');
            if (!tbody) {
                window.location.reload();
                return;
            }
            const row = document.createElement('tr');
            row.setAttribute('data-type', record.type);
            row.setAttribute('data-record-id', record.id);
            row.className = 'hover:bg-gray-50 border-b border-gray-100 bg-green-50';
            row.innerHTML = ` + "`" + `
                <td class="px-5 py-3 font-mono text-sm" data-field="name">${record.name}</td>
                <td class="px-5 py-3" data-field="type">${getTypeBadge(record.type)}</td>
                <td class="px-5 py-3 font-mono text-sm text-gray-600" data-field="value">${record.value}</td>
                <td class="px-5 py-3 text-gray-500" data-field="ttl">${record.ttl}</td>
                <td class="px-5 py-3">
                    <div class="flex gap-1">
                        <button onclick="showEditRecordModal(${record.id}, this)" class="text-blue-500 hover:text-blue-700" title="Edit">✏️</button>
                        <button onclick="deleteRecord(${record.id}, this)" class="text-red-500 hover:text-red-700" title="Delete">🗑</button>
                    </div>
                </td>
            ` + "`" + `;
            tbody.appendChild(row);
            setTimeout(() => row.classList.remove('bg-green-50'), 2000);
        }
        
        function getTypeBadge(type) {
            const colors = {
                'A': 'bg-blue-100 text-blue-700',
                'AAAA': 'bg-green-100 text-green-700',
                'CNAME': 'bg-orange-100 text-orange-700',
                'MX': 'bg-pink-100 text-pink-700',
                'TXT': 'bg-purple-100 text-purple-700',
                'NS': 'bg-teal-100 text-teal-700',
                'SOA': 'bg-gray-200 text-gray-700'
            };
            const color = colors[type] || 'bg-gray-100 text-gray-600';
            return '<span class="px-2 py-0.5 rounded text-sm font-medium ' + color + '">' + type + '</span>';
        }
        
        function showEditRecordModal(id, btn) {
            const row = btn.closest('tr');
            document.getElementById('editRecordId').value = id;
            document.getElementById('editRecordName').value = row.querySelector('[data-field="name"]').textContent;
            document.getElementById('editRecordType').value = row.querySelector('[data-field="type"]').textContent.trim();
            document.getElementById('editRecordValue').value = row.querySelector('[data-field="value"]').textContent;
            document.getElementById('editRecordTTL').value = row.querySelector('[data-field="ttl"]').textContent;
            document.getElementById('editRecordModal').classList.remove('hidden');
            document.getElementById('editRecordModal').classList.add('flex');
        }
        
        function hideEditRecordModal() {
            document.getElementById('editRecordModal').classList.add('hidden');
            document.getElementById('editRecordModal').classList.remove('flex');
        }
        
        async function submitEditRecord(event) {
            event.preventDefault();
            const id = document.getElementById('editRecordId').value;
            const data = {
                name: document.getElementById('editRecordName').value,
                type: document.getElementById('editRecordType').value,
                value: document.getElementById('editRecordValue').value,
                ttl: parseInt(document.getElementById('editRecordTTL').value) || 3600
            };
            try {
                const resp = await fetch('/api/records/' + id, {
                    method: 'PUT',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(data)
                });
                if (resp.ok) {
                    const row = document.querySelector('tr[data-record-id="' + id + '"]');
                    if (row) {
                        row.querySelector('[data-field="name"]').textContent = data.name;
                        row.querySelector('[data-field="type"]').innerHTML = getTypeBadge(data.type);
                        row.querySelector('[data-field="value"]').textContent = data.value;
                        row.querySelector('[data-field="ttl"]').textContent = data.ttl;
                        row.setAttribute('data-type', data.type);
                    }
                    hideEditRecordModal();
                } else {
                    const err = await resp.json();
                    alert('Failed to update record: ' + (err.error || 'Unknown error'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
        
        async function deleteRecord(id, btn) {
            if (!confirm('Delete this record?')) return;
            try {
                const resp = await fetch('/api/records/' + id, { method: 'DELETE' });
                if (resp.ok) {
                    btn.closest('tr').remove();
                } else {
                    alert('Failed to delete record');
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
    </script>
</body>
</html>
`

// Zone Settings page
const zoneSettingsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SimpleDNS - {{.Zone.Name}} Settings</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        primary: '#3498db',
                        'primary-dark': '#2980b9',
                    }
                }
            }
        }
    </script>
</head>
<body class="bg-gray-100 text-gray-800 font-sans">
    <div class="max-w-6xl mx-auto p-5">
        <!-- Header -->
        <header class="bg-slate-800 text-white p-5 mb-5 rounded-lg">
            <div class="flex justify-between items-start">
                <div>
                    <div class="flex items-center gap-2 mb-2">
                        <a href="/" class="text-white/70 hover:text-white">🏠 Home</a>
                        <span class="text-white/50">/</span>
                        <span>{{.Zone.Name}}</span>
                    </div>
                    <h1 class="text-2xl font-bold">⚙️ Zone Settings</h1>
                    <div class="flex gap-4 mt-3">
                        <span class="bg-white/10 px-4 py-2 rounded">Zone: {{.Zone.Name}}</span>
                    </div>
                </div>
                <a href="/zones/{{.Zone.Name}}/records" class="bg-primary hover:bg-primary-dark px-4 py-2 rounded font-medium transition-colors">📝 Records</a>
            </div>
        </header>

        <!-- Zone Info -->
        <div class="bg-white rounded-lg shadow mb-5">
            <div class="px-5 py-4 border-b">
                <h2 class="text-lg font-bold text-gray-700">Zone Information</h2>
            </div>
            <div class="p-5">
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <label class="block text-sm font-medium text-gray-500">Zone Name</label>
                        <p class="text-lg font-mono">{{.Zone.Name}}</p>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-500">Records Count</label>
                        <p class="text-lg">{{len .Zone.Records}}</p>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-500">Zone ID</label>
                        <p class="text-lg font-mono">{{.Zone.ID}}</p>
                    </div>
                    <div>
                        <label class="block text-sm font-medium text-gray-500">Mode</label>
                        <p class="text-lg">{{.Mode}}</p>
                    </div>
                </div>
            </div>
        </div>

        {{if .EditMode}}
        <!-- Danger Zone -->
        <div class="bg-white rounded-lg shadow border-2 border-red-200">
            <div class="px-5 py-4 border-b bg-red-50">
                <h2 class="text-lg font-bold text-red-700">⚠️ Danger Zone</h2>
            </div>
            <div class="p-5">
                <div class="flex justify-between items-center">
                    <div>
                        <h3 class="font-medium text-gray-800">Delete this zone</h3>
                        <p class="text-sm text-gray-500">Once deleted, all records in this zone will be permanently removed.</p>
                    </div>
                    <button onclick="deleteZone()" class="bg-red-500 hover:bg-red-600 text-white px-4 py-2 rounded transition-colors">Delete Zone</button>
                </div>
            </div>
        </div>
        {{else}}
        <div class="bg-white rounded-lg shadow">
            <div class="p-5 text-center text-gray-400">
                Zone management is read-only in files mode.
            </div>
        </div>
        {{end}}

        <!-- Footer -->
        <footer class="text-center py-5 text-gray-400 text-sm">
            <a href="/" class="text-primary hover:underline">← Back to Dashboard</a>
        </footer>
    </div>
    
    <script>
        const zoneId = {{.Zone.ID}};
        const zoneName = '{{.Zone.Name}}';
        
        async function deleteZone() {
            if (!confirm('Are you sure you want to delete zone ' + zoneName + '? This will remove all records and cannot be undone.')) return;
            if (!confirm('This is your last chance. Are you really sure?')) return;
            
            try {
                const resp = await fetch('/api/zones/' + zoneId, { method: 'DELETE' });
                if (resp.ok) {
                    window.location.href = '/';
                } else {
                    const err = await resp.json();
                    alert('Failed to delete zone: ' + (err.error || 'Unknown error'));
                }
            } catch(e) {
                alert('Error: ' + e.message);
            }
        }
    </script>
</body>
</html>
`
