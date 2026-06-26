import { el, icon, clear } from "../core/dom.ts";
import { attachContextMenu } from "./ContextMenu.ts";
import { type MenuItem } from "./Menu.ts";
import { notify } from "./Toast.ts";

export interface GridColumn {
  name: string;
  type?: string;
  primary?: boolean;
  /** When set, the cell value renders as a link (e.g. a foreign key). */
  link?: (value: string) => void;
}

export interface DataGridOptions {
  columns: GridColumn[];
  rows: Array<Array<string | null>>;
  sortBy?: string;
  sortDir?: "asc" | "desc";
  rowNumberStart?: number;
  onSort?: (column: string) => void;
  rowMenu?: (rowIndex: number, row: Array<string | null>) => MenuItem[];
}

export interface DataGridHandle {
  el: HTMLElement;
  setData(columns: GridColumn[], rows: Array<Array<string | null>>): void;
}

const NUMERIC = /^(INT|BIGINT|SMALLINT|TINYINT|MEDIUMINT|DECIMAL|FLOAT|DOUBLE|YEAR|BIT)/i;

/** Spreadsheet-style result grid: sticky header, NULL styling, cell copy. */
export function DataGrid(opts: DataGridOptions): DataGridHandle {
  const table = el("table.gtma-grid");
  const root = el("div.gtma-grid__scroll", {}, table);

  function render(columns: GridColumn[], rows: Array<Array<string | null>>, sortBy?: string, sortDir?: string) {
    clear(table);

    // Header.
    const headRow = el("tr", {},
      el("th.gtma-grid__rownum", {}, "#"),
      ...columns.map((c) => {
        const sortable = !!opts.onSort;
        const isSorted = sortBy === c.name;
        return el("th.gtma-grid__th", {
          class: [sortable ? "is-sortable" : "", isSorted ? "is-sorted" : ""].join(" "),
          onclick: sortable ? () => opts.onSort!(c.name) : undefined,
        },
          el("div.gtma-grid__th-inner", {},
            c.primary ? icon("key", { class: "gtma-grid__pk" }) : null,
            el("span.gtma-grid__col-name", {}, c.name),
            c.type ? el("span.gtma-grid__col-type.mono", {}, baseType(c.type)) : null,
            isSorted ? icon(sortDir === "desc" ? "arrow-down-long" : "arrow-up-long", { class: "gtma-grid__sort" }) : null,
          ),
        );
      }),
    );
    table.appendChild(el("thead", {}, headRow));

    // Body.
    const tbody = el("tbody");
    const start = opts.rowNumberStart ?? 1;
    rows.forEach((row, ri) => {
      const tr = el("tr.gtma-grid__row", {},
        el("td.gtma-grid__rownum", {}, String(start + ri)),
        ...row.map((cell, ci) => renderCell(cell, columns[ci])),
      );
      if (opts.rowMenu) attachContextMenu(tr, () => opts.rowMenu!(ri, row));
      tbody.appendChild(tr);
    });
    table.appendChild(tbody);

    if (!rows.length) {
      table.appendChild(el("tbody", {}, el("tr", {},
        el("td.gtma-grid__empty", { attrs: { colspan: columns.length + 1 } }, "No rows"))));
    }
  }

  function renderCell(cell: string | null, col: GridColumn): HTMLElement {
    if (cell === null) {
      return el("td.gtma-grid__cell.gtma-grid__cell--null", { attrs: { title: "NULL" } }, "NULL");
    }
    const numeric = col?.type ? NUMERIC.test(col.type) : false;
    const truncated = cell.length > 220 ? cell.slice(0, 220) + "…" : cell;
    if (col?.link) {
      return el("td.gtma-grid__cell", { class: numeric ? "gtma-grid__cell--num" : "" },
        el("button.gtma-grid__fk", {
          attrs: { type: "button", title: `Jump to referenced row (${cell})` },
          onclick: (e: MouseEvent) => { e.stopPropagation(); col.link!(cell); },
        }, el("span.mono.truncate", {}, truncated), icon("arrow-up-right-from-square")));
    }
    return el("td.gtma-grid__cell", {
      class: numeric ? "gtma-grid__cell--num mono" : "",
      attrs: { title: cell.length > 80 ? cell : null },
      ondblclick: () => copyCell(cell),
    }, truncated);
  }

  async function copyCell(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      notify.success("Cell copied");
    } catch {
      notify.error("Clipboard unavailable");
    }
  }

  render(opts.columns, opts.rows, opts.sortBy, opts.sortDir);

  return {
    el: root,
    setData: (columns, rows) => render(columns, rows, opts.sortBy, opts.sortDir),
  };
}

function baseType(t: string): string {
  return t.replace(/\(.*$/, "").toLowerCase();
}
