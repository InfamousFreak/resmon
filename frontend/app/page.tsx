"use client";

import { useEffect, useState, type KeyboardEvent } from "react";

// The same struct from your Go backend!
interface Paper {
  arxiv_id: string;
  title: string;
  abstract: string;
  efficiency_score: number;
  speedup: string;
  hardware: string;
  is_implementable: boolean;
  github_url: string | null;
  edge_type?: string;
  is_theoretical?: boolean;
}

export default function Dashboard() {
  const [papers, setPapers] = useState<Paper[]>([]);
  const [loading, setLoading] = useState(true);
  const [lastSync, setLastSync] = useState("--:--:--");
  const [searchQuery, setSearchQuery] = useState("");

  const fetchLatest = (showLoader = true) => {
    if (showLoader) {
      setLoading(true);
    }
    fetch("http://localhost:8080/api/papers")
      .then((res) => res.json())
      .then((data) => {
        setPapers(data || []);
        setLastSync(new Date().toLocaleTimeString());
        setLoading(false);
      })
      .catch((err) => {
        console.error("Failed to fetch papers:", err);
        setLastSync(new Date().toLocaleTimeString());
        setLoading(false);
      });
  };

  useEffect(() => {
    fetch("http://localhost:8080/api/papers")
      .then((res) => res.json())
      .then((data) => {
        setPapers(data || []);
        setLastSync(new Date().toLocaleTimeString());
        setLoading(false);
      })
      .catch((err) => {
        console.error("Failed to fetch papers:", err);
        setLastSync(new Date().toLocaleTimeString());
        setLoading(false);
      });
  }, []);

  const handleSearch = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== "Enter") {
      return;
    }

    if (searchQuery.trim() === "") {
      fetchLatest();
      return;
    }

    setLoading(true);
    fetch(`http://localhost:8080/api/search?q=${encodeURIComponent(searchQuery)}`)
      .then((res) => res.json())
      .then((data) => {
        setPapers(data || []);
        setLastSync(new Date().toLocaleTimeString());
        setLoading(false);
      })
      .catch((err) => {
        console.error("Failed to search papers:", err);
        setLastSync(new Date().toLocaleTimeString());
        setLoading(false);
      });
  };

  return (
    <main className="min-h-screen bg-black text-green-500 font-mono pb-16">
      {/* Top Header */}
      <header className="p-4 border-b border-green-900/50 bg-black sticky top-0 z-10 shadow-[0_4px_20px_rgba(0,0,0,0.8)] flex justify-between items-end flex-wrap gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tighter flex items-center gap-2">
            <span className="w-2 h-2 bg-red-500 rounded-full animate-pulse"></span>
            RESEARCH_MONITOR_v1.0
          </h1>
          <p className="text-green-700 text-xs mt-1">
            STATUS: ACTIVE | ENGINE: GO_ROUTER | LLM: GEMMA-4B
          </p>
        </div>

        <div className="flex-1 min-w-[300px] max-w-xl mx-4">
          <div className="relative flex items-center bg-black border border-green-900 focus-within:border-green-400 focus-within:shadow-[0_0_15px_rgba(34,197,94,0.2)] transition-all">
            <span className="pl-3 text-green-500 font-bold">&gt;</span>
            <input
              type="text"
              className="w-full bg-transparent p-2 text-green-400 outline-none placeholder-green-900 text-sm"
              placeholder="EXECUTE_QUERY: (e.g., 'low memory LLM inference')"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={handleSearch}
            />
          </div>
        </div>

        <div className="text-right text-xs text-green-700">
          <p>TOTAL_NODES: {papers.length}</p>
          <p>LAST_SYNC: {lastSync}</p>
        </div>
      </header>

      {/* Main Grid */}
      <div className="p-4">
        {loading ? (
          <div className="text-sm animate-pulse">&gt; Initializing databanks...</div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-3">
            {papers.length === 0 ? (
              <div className="col-span-full flex flex-col items-center justify-center py-24 text-red-500/80 border border-red-900/30 bg-[#050000] shadow-[inset_0_0_20px_rgba(220,38,38,0.1)]">
                <span className="text-4xl mb-4 animate-pulse">_</span>
                <p className="text-sm font-bold tracking-widest">[ ERR: 0_NODES_FOUND ]</p>
                <p className="text-xs mt-2 text-red-700">QUERY EXCEEDS MATHEMATICAL THRESHOLD. NO RELEVANT INTEL EXISTS.</p>
                <button
                  onClick={() => fetchLatest()}
                  className="mt-6 px-4 py-2 border border-red-900/50 text-[10px] hover:bg-red-900/20 hover:text-red-400 transition-colors"
                >
                  [ RESET_DATABANKS ]
                </button>
              </div>
            ) : (
              papers.map((paper) => {
                const edgeType = (paper.edge_type || "general").toUpperCase();
                const isTheoretical = Boolean(paper.is_theoretical);

                return (
                  <div
                    key={paper.arxiv_id}
                    className="relative flex flex-col border border-green-900/40 bg-[#050505] p-3 hover:border-green-500 hover:bg-[#0a0a0a] transition-all duration-200"
                  >
                    {/* The "Edge" Ribbon */}
                    <div className="absolute top-0 right-0 bg-green-900/40 text-[8px] px-1 uppercase tracking-widest text-green-300 border-l border-b border-green-500/30">
                      {paper.edge_type || "General"} Optimized
                    </div>

                    {/* ID & Score */}
                    <div className="flex justify-between items-start mb-2 pb-2 border-b border-green-900/30">
                      <span className="text-[10px] text-green-800 truncate pr-2" title={paper.arxiv_id}>
                        {paper.arxiv_id.split("v")[0]}
                      </span>
                      <span
                        className={`text-[10px] font-bold px-1.5 py-0.5 ${paper.efficiency_score > 70 ? "bg-green-900 text-green-300" : "text-gray-600"}`}
                      >
                        SCORE:{paper.efficiency_score}
                      </span>
                    </div>

                    {/* Title */}
                    <h2 className="text-xs font-bold mb-3 line-clamp-3 leading-relaxed text-gray-200">
                      {paper.title}
                    </h2>

                    {/* Metrics */}
                    <div className="text-[10px] space-y-1.5 mt-auto bg-black/50 p-2 border border-green-900/20">
                      <div className="flex justify-between">
                        <span className="text-green-800">SPEEDUP:</span>
                        <span className="text-gray-400 text-right truncate w-24">{paper.speedup || "N/A"}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-green-800">HARDWARE:</span>
                        <span className="text-gray-400 text-right truncate w-24">{paper.hardware || "N/A"}</span>
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-2 mt-2 text-[10px]">
                      <div
                        className={`p-1 border ${isTheoretical ? "border-red-900/30 text-red-900" : "border-green-900/30 text-green-500"
                          }`}
                      >
                        STATUS: {isTheoretical ? "THEORETICAL" : "EMPIRICAL"}
                      </div>
                      <div className="p-1 border border-green-900/30 text-green-500">EDGE: {edgeType}</div>
                    </div>

                    {/* Cyberpunk Button */}
                    {paper.github_url ? (
                      <a
                        href={paper.github_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="mt-3 block w-full text-center text-[10px] font-bold py-1.5 bg-green-900/20 text-green-400 border border-green-500/50 hover:bg-green-500 hover:text-black hover:shadow-[0_0_10px_rgba(34,197,94,0.4)] transition-all"
                      >
                        [&lt;/&gt; GET_CODE]
                      </a>
                    ) : (
                      <div className="mt-3 block w-full text-center text-[10px] py-1.5 text-green-900 border border-green-900/30">
                        [ NO_REPO_FOUND ]
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>
        )}
      </div>

      {/* Live Ticker Footer */}
      <footer className="fixed bottom-0 left-0 w-full bg-black border-t border-green-900 py-1 overflow-hidden z-20">
        <div className="animate-ticker text-[10px] text-green-500 font-bold tracking-widest flex gap-8">
          <span>&gt; LATEST INTEL: </span>
          {papers.filter(p => p.efficiency_score > 50).map(p => (
            <span key={`ticker-${p.arxiv_id}`}>
              [{p.speedup && p.speedup !== "None" ? `SPEEDUP: ${p.speedup} | ` : ""}{p.title.toUpperCase()}] ///
            </span>
          ))}
        </div>
      </footer>
    </main>
  );
}