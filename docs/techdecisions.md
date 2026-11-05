# Technical Decisions

This document outlines the key architectural decisions made during the development of Kaze, justifying how these choices align with the project's core goals and constraints.

## Core Philosophy: Minimalist & Lightweight
The primary goal of Kaze is to provide a functional, minimal music server that runs efficiently on extremely limited hardware, specifically targeting platforms like the Raspberry Pi 3B.

### Decision 1: No Database Layer
*   **Justification:** Traditional database solutions (e.g., SQLite, MySQL) require significant I/O and memory overhead. For a Raspberry Pi 3B, running a database alongside the core application would introduce unnecessary complexity and memory pressure. By opting for an in-memory index built directly from the filesystem, startup time might be longer, but runtime overhead is minimized to near zero.

### Decision 2: Filesystem-First Data Model (Folder Structure)
*   **Justification:** Kaze relies entirely on the expected directory structure (`Artist/Album/Track`) for data retrieval. This avoids the need to define, manage, and query a separate database schema. The filesystem acts as the "single source of truth." This simplifies the data model and dramatically reduces complexity, allowing the application to be extremely lean.

### Decision 3: Limited/Subset Subsonic API Implementation
*   **Justification:** Implementing the *entire* Subsonic API is a massive undertaking and would introduce significant code volume and complexity. Kaze focuses only on the endpoints strictly necessary for basic music playback and library browsing (`ping`, `getMusicFolders`, `stream`, etc.). This ensures the codebase remains small, maintainable, and lightweight, fulfilling the "get rid of everything non-essential" mandate.

### Decision 4: Minimal Security Implementation
*   **Justification:** The initial focus is on functionality and minimal resource usage, not enterprise-grade security. Security measures (like authentication and authorization) are implemented only to the bare minimum required to prevent unauthorized access to the library, but they are intentionally lightweight and non-robust. This is a deliberate trade-off to prioritize performance and resource efficiency on low-power hardware.

## Summary of Trade-offs
| Decision | Benefit | Trade-off |
| :--- | :--- | :--- |
| In-Memory Index | Extremely fast runtime lookups; Zero I/O during serving. | Longer startup time during initial scan; Data is lost on power loss/crash. |
| Filesystem-First | Minimal code complexity; No database overhead. | Requires strict adherence to directory naming conventions; Data querying is limited to filesystem traversal. |
| Limited API Scope | Extremely small codebase; Fast deployment. | Clients must only support the implemented endpoints; Some advanced Subsonic features are unavailable. |
| Minimal Security | Lowest possible resource usage. | Security is limited and may not meet high-security enterprise standards. |

These decisions collectively ensure Kaze remains a highly optimized, resource-friendly music server tailored for the constraints of limited hardware.

Do you require further clarification on any of these points?