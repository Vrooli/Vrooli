export const feed_home = {
  "fieldName": "home",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "home",
        "loc": {
          "start": 2737,
          "end": 2741
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2742,
              "end": 2747
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 2750,
                "end": 2755
              }
            },
            "loc": {
              "start": 2749,
              "end": 2755
            }
          },
          "loc": {
            "start": 2742,
            "end": 2755
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notes",
              "loc": {
                "start": 2763,
                "end": 2768
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Note_list",
                    "loc": {
                      "start": 2782,
                      "end": 2791
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2779,
                    "end": 2791
                  }
                }
              ],
              "loc": {
                "start": 2769,
                "end": 2797
              }
            },
            "loc": {
              "start": 2763,
              "end": 2797
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminders",
              "loc": {
                "start": 2802,
                "end": 2811
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Reminder_full",
                    "loc": {
                      "start": 2825,
                      "end": 2838
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2822,
                    "end": 2838
                  }
                }
              ],
              "loc": {
                "start": 2812,
                "end": 2844
              }
            },
            "loc": {
              "start": 2802,
              "end": 2844
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resources",
              "loc": {
                "start": 2849,
                "end": 2858
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Resource_list",
                    "loc": {
                      "start": 2872,
                      "end": 2885
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2869,
                    "end": 2885
                  }
                }
              ],
              "loc": {
                "start": 2859,
                "end": 2891
              }
            },
            "loc": {
              "start": 2849,
              "end": 2891
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedules",
              "loc": {
                "start": 2896,
                "end": 2905
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Schedule_list",
                    "loc": {
                      "start": 2919,
                      "end": 2932
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2916,
                    "end": 2932
                  }
                }
              ],
              "loc": {
                "start": 2906,
                "end": 2938
              }
            },
            "loc": {
              "start": 2896,
              "end": 2938
            }
          }
        ],
        "loc": {
          "start": 2757,
          "end": 2942
        }
      },
      "loc": {
        "start": 2737,
        "end": 2942
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 32,
          "end": 34
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 32,
        "end": 34
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 35,
          "end": 45
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 35,
        "end": 45
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 46,
          "end": 56
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 46,
        "end": 56
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 57,
          "end": 62
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 57,
        "end": 62
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 63,
          "end": 68
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 63,
        "end": 68
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 69,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 88,
                  "end": 100
                }
              },
              "loc": {
                "start": 88,
                "end": 100
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 114,
                      "end": 130
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 111,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 101,
                "end": 136
              }
            },
            "loc": {
              "start": 81,
              "end": 136
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 148,
                  "end": 152
                }
              },
              "loc": {
                "start": 148,
                "end": 152
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 166,
                      "end": 174
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 174
                  }
                }
              ],
              "loc": {
                "start": 153,
                "end": 180
              }
            },
            "loc": {
              "start": 141,
              "end": 180
            }
          }
        ],
        "loc": {
          "start": 75,
          "end": 182
        }
      },
      "loc": {
        "start": 69,
        "end": 182
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 183,
          "end": 186
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 193,
                "end": 202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 193,
              "end": 202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 207,
                "end": 216
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 207,
              "end": 216
            }
          }
        ],
        "loc": {
          "start": 187,
          "end": 218
        }
      },
      "loc": {
        "start": 183,
        "end": 218
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 250,
          "end": 258
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 265,
                "end": 277
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 288,
                      "end": 290
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 288,
                    "end": 290
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 299,
                      "end": 307
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 299,
                    "end": 307
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 316,
                      "end": 327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 316,
                    "end": 327
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 336,
                      "end": 340
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 336,
                    "end": 340
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "text",
                    "loc": {
                      "start": 349,
                      "end": 353
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 349,
                    "end": 353
                  }
                }
              ],
              "loc": {
                "start": 278,
                "end": 359
              }
            },
            "loc": {
              "start": 265,
              "end": 359
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 364,
                "end": 366
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 364,
              "end": 366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 371,
                "end": 381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 371,
              "end": 381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 386,
                "end": 396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 386,
              "end": 396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 401,
                "end": 409
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 401,
              "end": 409
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 414,
                "end": 423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 414,
              "end": 423
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 428,
                "end": 440
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 428,
              "end": 440
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 445,
                "end": 457
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 445,
              "end": 457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 462,
                "end": 474
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 462,
              "end": 474
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 479,
                "end": 482
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 493,
                      "end": 503
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 493,
                    "end": 503
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 512,
                      "end": 519
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 512,
                    "end": 519
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 528,
                      "end": 537
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 528,
                    "end": 537
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 546,
                      "end": 555
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 546,
                    "end": 555
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 564,
                      "end": 573
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 564,
                    "end": 573
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 582,
                      "end": 588
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 582,
                    "end": 588
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 597,
                      "end": 604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 597,
                    "end": 604
                  }
                }
              ],
              "loc": {
                "start": 483,
                "end": 610
              }
            },
            "loc": {
              "start": 479,
              "end": 610
            }
          }
        ],
        "loc": {
          "start": 259,
          "end": 612
        }
      },
      "loc": {
        "start": 250,
        "end": 612
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 613,
          "end": 615
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 613,
        "end": 615
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 616,
          "end": 626
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 616,
        "end": 626
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 627,
          "end": 637
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 627,
        "end": 637
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 638,
          "end": 647
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 638,
        "end": 647
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 648,
          "end": 659
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 648,
        "end": 659
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 660,
          "end": 666
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 676,
                "end": 686
              }
            },
            "directives": [],
            "loc": {
              "start": 673,
              "end": 686
            }
          }
        ],
        "loc": {
          "start": 667,
          "end": 688
        }
      },
      "loc": {
        "start": 660,
        "end": 688
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 689,
          "end": 694
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 708,
                  "end": 720
                }
              },
              "loc": {
                "start": 708,
                "end": 720
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 734,
                      "end": 750
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 731,
                    "end": 750
                  }
                }
              ],
              "loc": {
                "start": 721,
                "end": 756
              }
            },
            "loc": {
              "start": 701,
              "end": 756
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 768,
                  "end": 772
                }
              },
              "loc": {
                "start": 768,
                "end": 772
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 786,
                      "end": 794
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 783,
                    "end": 794
                  }
                }
              ],
              "loc": {
                "start": 773,
                "end": 800
              }
            },
            "loc": {
              "start": 761,
              "end": 800
            }
          }
        ],
        "loc": {
          "start": 695,
          "end": 802
        }
      },
      "loc": {
        "start": 689,
        "end": 802
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 803,
          "end": 814
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 803,
        "end": 814
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 815,
          "end": 829
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 815,
        "end": 829
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 830,
          "end": 835
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 830,
        "end": 835
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 836,
          "end": 845
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 836,
        "end": 845
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 846,
          "end": 850
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 860,
                "end": 868
              }
            },
            "directives": [],
            "loc": {
              "start": 857,
              "end": 868
            }
          }
        ],
        "loc": {
          "start": 851,
          "end": 870
        }
      },
      "loc": {
        "start": 846,
        "end": 870
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 871,
          "end": 885
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 871,
        "end": 885
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 886,
          "end": 891
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 886,
        "end": 891
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 892,
          "end": 895
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 902,
                "end": 911
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 902,
              "end": 911
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 916,
                "end": 927
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 916,
              "end": 927
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 932,
                "end": 943
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 932,
              "end": 943
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 948,
                "end": 957
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 948,
              "end": 957
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 962,
                "end": 969
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 962,
              "end": 969
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 974,
                "end": 982
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 974,
              "end": 982
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 987,
                "end": 999
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 987,
              "end": 999
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1004,
                "end": 1012
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1004,
              "end": 1012
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1017,
                "end": 1025
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1017,
              "end": 1025
            }
          }
        ],
        "loc": {
          "start": 896,
          "end": 1027
        }
      },
      "loc": {
        "start": 892,
        "end": 1027
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1074,
          "end": 1076
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1074,
        "end": 1076
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1077,
          "end": 1083
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1077,
        "end": 1083
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1084,
          "end": 1087
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 1094,
                "end": 1107
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1094,
              "end": 1107
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1112,
                "end": 1121
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1112,
              "end": 1121
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1126,
                "end": 1137
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1126,
              "end": 1137
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 1142,
                "end": 1151
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1142,
              "end": 1151
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1156,
                "end": 1165
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1156,
              "end": 1165
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1170,
                "end": 1177
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1170,
              "end": 1177
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1182,
                "end": 1194
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1182,
              "end": 1194
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1199,
                "end": 1207
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1199,
              "end": 1207
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 1212,
                "end": 1226
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1237,
                      "end": 1239
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1237,
                    "end": 1239
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1248,
                      "end": 1258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1248,
                    "end": 1258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1267,
                      "end": 1277
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1267,
                    "end": 1277
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 1286,
                      "end": 1293
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1286,
                    "end": 1293
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 1302,
                      "end": 1313
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1302,
                    "end": 1313
                  }
                }
              ],
              "loc": {
                "start": 1227,
                "end": 1319
              }
            },
            "loc": {
              "start": 1212,
              "end": 1319
            }
          }
        ],
        "loc": {
          "start": 1088,
          "end": 1321
        }
      },
      "loc": {
        "start": 1084,
        "end": 1321
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1361,
          "end": 1363
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1361,
        "end": 1363
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1364,
          "end": 1374
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1364,
        "end": 1374
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1375,
          "end": 1385
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1375,
        "end": 1385
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 1386,
          "end": 1390
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1386,
        "end": 1390
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "description",
        "loc": {
          "start": 1391,
          "end": 1402
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1391,
        "end": 1402
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "dueDate",
        "loc": {
          "start": 1403,
          "end": 1410
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1403,
        "end": 1410
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 1411,
          "end": 1416
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1411,
        "end": 1416
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isComplete",
        "loc": {
          "start": 1417,
          "end": 1427
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1417,
        "end": 1427
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderItems",
        "loc": {
          "start": 1428,
          "end": 1441
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1448,
                "end": 1450
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1448,
              "end": 1450
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1455,
                "end": 1465
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1455,
              "end": 1465
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1470,
                "end": 1480
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1470,
              "end": 1480
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1485,
                "end": 1489
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1485,
              "end": 1489
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1494,
                "end": 1505
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1494,
              "end": 1505
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 1510,
                "end": 1517
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1510,
              "end": 1517
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1522,
                "end": 1527
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1522,
              "end": 1527
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1532,
                "end": 1542
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1532,
              "end": 1542
            }
          }
        ],
        "loc": {
          "start": 1442,
          "end": 1544
        }
      },
      "loc": {
        "start": 1428,
        "end": 1544
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderList",
        "loc": {
          "start": 1545,
          "end": 1557
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1564,
                "end": 1566
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1564,
              "end": 1566
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1571,
                "end": 1581
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1571,
              "end": 1581
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1586,
                "end": 1596
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1586,
              "end": 1596
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusMode",
              "loc": {
                "start": 1601,
                "end": 1610
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 1621,
                      "end": 1627
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1642,
                            "end": 1644
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1642,
                          "end": 1644
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 1657,
                            "end": 1662
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1657,
                          "end": 1662
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 1675,
                            "end": 1680
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1675,
                          "end": 1680
                        }
                      }
                    ],
                    "loc": {
                      "start": 1628,
                      "end": 1690
                    }
                  },
                  "loc": {
                    "start": 1621,
                    "end": 1690
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "schedule",
                    "loc": {
                      "start": 1699,
                      "end": 1707
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Schedule_common",
                          "loc": {
                            "start": 1725,
                            "end": 1740
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1722,
                          "end": 1740
                        }
                      }
                    ],
                    "loc": {
                      "start": 1708,
                      "end": 1750
                    }
                  },
                  "loc": {
                    "start": 1699,
                    "end": 1750
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1759,
                      "end": 1761
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1759,
                    "end": 1761
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1770,
                      "end": 1774
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1770,
                    "end": 1774
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1783,
                      "end": 1794
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1783,
                    "end": 1794
                  }
                }
              ],
              "loc": {
                "start": 1611,
                "end": 1800
              }
            },
            "loc": {
              "start": 1601,
              "end": 1800
            }
          }
        ],
        "loc": {
          "start": 1558,
          "end": 1802
        }
      },
      "loc": {
        "start": 1545,
        "end": 1802
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1842,
          "end": 1844
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1842,
        "end": 1844
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 1845,
          "end": 1850
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1845,
        "end": 1850
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "link",
        "loc": {
          "start": 1851,
          "end": 1855
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1851,
        "end": 1855
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "usedFor",
        "loc": {
          "start": 1856,
          "end": 1863
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1856,
        "end": 1863
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 1864,
          "end": 1876
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1883,
                "end": 1885
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1883,
              "end": 1885
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 1890,
                "end": 1898
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1890,
              "end": 1898
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1903,
                "end": 1914
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1903,
              "end": 1914
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1919,
                "end": 1923
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1919,
              "end": 1923
            }
          }
        ],
        "loc": {
          "start": 1877,
          "end": 1925
        }
      },
      "loc": {
        "start": 1864,
        "end": 1925
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1967,
          "end": 1969
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1967,
        "end": 1969
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1970,
          "end": 1980
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1970,
        "end": 1980
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1981,
          "end": 1991
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1981,
        "end": 1991
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 1992,
          "end": 2001
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1992,
        "end": 2001
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 2002,
          "end": 2009
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2002,
        "end": 2009
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 2010,
          "end": 2018
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2010,
        "end": 2018
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 2019,
          "end": 2029
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2036,
                "end": 2038
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2036,
              "end": 2038
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 2043,
                "end": 2060
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2043,
              "end": 2060
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 2065,
                "end": 2077
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2065,
              "end": 2077
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 2082,
                "end": 2092
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2082,
              "end": 2092
            }
          }
        ],
        "loc": {
          "start": 2030,
          "end": 2094
        }
      },
      "loc": {
        "start": 2019,
        "end": 2094
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 2095,
          "end": 2106
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2113,
                "end": 2115
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2113,
              "end": 2115
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 2120,
                "end": 2134
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2120,
              "end": 2134
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 2139,
                "end": 2147
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2139,
              "end": 2147
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 2152,
                "end": 2161
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2152,
              "end": 2161
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 2166,
                "end": 2176
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2166,
              "end": 2176
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 2181,
                "end": 2186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2181,
              "end": 2186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 2191,
                "end": 2198
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2191,
              "end": 2198
            }
          }
        ],
        "loc": {
          "start": 2107,
          "end": 2200
        }
      },
      "loc": {
        "start": 2095,
        "end": 2200
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 2240,
          "end": 2246
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 2256,
                "end": 2266
              }
            },
            "directives": [],
            "loc": {
              "start": 2253,
              "end": 2266
            }
          }
        ],
        "loc": {
          "start": 2247,
          "end": 2268
        }
      },
      "loc": {
        "start": 2240,
        "end": 2268
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2269,
          "end": 2271
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2269,
        "end": 2271
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2272,
          "end": 2282
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2272,
        "end": 2282
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2283,
          "end": 2293
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2283,
        "end": 2293
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 2294,
          "end": 2303
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2294,
        "end": 2303
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 2304,
          "end": 2311
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2304,
        "end": 2311
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 2312,
          "end": 2320
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2312,
        "end": 2320
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 2321,
          "end": 2331
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2338,
                "end": 2340
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2338,
              "end": 2340
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 2345,
                "end": 2362
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2345,
              "end": 2362
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 2367,
                "end": 2379
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2367,
              "end": 2379
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 2384,
                "end": 2394
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2384,
              "end": 2394
            }
          }
        ],
        "loc": {
          "start": 2332,
          "end": 2396
        }
      },
      "loc": {
        "start": 2321,
        "end": 2396
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 2397,
          "end": 2408
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2415,
                "end": 2417
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2415,
              "end": 2417
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 2422,
                "end": 2436
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2422,
              "end": 2436
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 2441,
                "end": 2449
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2441,
              "end": 2449
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 2454,
                "end": 2463
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2454,
              "end": 2463
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 2468,
                "end": 2478
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2468,
              "end": 2478
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 2483,
                "end": 2488
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2483,
              "end": 2488
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 2493,
                "end": 2500
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2493,
              "end": 2500
            }
          }
        ],
        "loc": {
          "start": 2409,
          "end": 2502
        }
      },
      "loc": {
        "start": 2397,
        "end": 2502
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2532,
          "end": 2534
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2532,
        "end": 2534
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2535,
          "end": 2545
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2535,
        "end": 2545
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 2546,
          "end": 2549
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2546,
        "end": 2549
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2550,
          "end": 2559
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2550,
        "end": 2559
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 2560,
          "end": 2572
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2579,
                "end": 2581
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2579,
              "end": 2581
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 2586,
                "end": 2594
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2586,
              "end": 2594
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 2599,
                "end": 2610
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2599,
              "end": 2610
            }
          }
        ],
        "loc": {
          "start": 2573,
          "end": 2612
        }
      },
      "loc": {
        "start": 2560,
        "end": 2612
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2613,
          "end": 2616
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOwn",
              "loc": {
                "start": 2623,
                "end": 2628
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2623,
              "end": 2628
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2633,
                "end": 2645
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2633,
              "end": 2645
            }
          }
        ],
        "loc": {
          "start": 2617,
          "end": 2647
        }
      },
      "loc": {
        "start": 2613,
        "end": 2647
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2678,
          "end": 2680
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2678,
        "end": 2680
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 2681,
          "end": 2686
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2681,
        "end": 2686
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 2687,
          "end": 2691
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2687,
        "end": 2691
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2692,
          "end": 2698
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2692,
        "end": 2698
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 10,
          "end": 20
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 24,
            "end": 29
          }
        },
        "loc": {
          "start": 24,
          "end": 29
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 32,
                "end": 34
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 32,
              "end": 34
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 35,
                "end": 45
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 35,
              "end": 45
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 46,
                "end": 56
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 46,
              "end": 56
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 57,
                "end": 62
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 57,
              "end": 62
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 63,
                "end": 68
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 63,
              "end": 68
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 69,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 88,
                        "end": 100
                      }
                    },
                    "loc": {
                      "start": 88,
                      "end": 100
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 114,
                            "end": 130
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 111,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 101,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 81,
                    "end": 136
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 148,
                        "end": 152
                      }
                    },
                    "loc": {
                      "start": 148,
                      "end": 152
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 166,
                            "end": 174
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 163,
                          "end": 174
                        }
                      }
                    ],
                    "loc": {
                      "start": 153,
                      "end": 180
                    }
                  },
                  "loc": {
                    "start": 141,
                    "end": 180
                  }
                }
              ],
              "loc": {
                "start": 75,
                "end": 182
              }
            },
            "loc": {
              "start": 69,
              "end": 182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 183,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 193,
                      "end": 202
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 193,
                    "end": 202
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 207,
                      "end": 216
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 207,
                    "end": 216
                  }
                }
              ],
              "loc": {
                "start": 187,
                "end": 218
              }
            },
            "loc": {
              "start": 183,
              "end": 218
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 220
        }
      },
      "loc": {
        "start": 1,
        "end": 220
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 230,
          "end": 239
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 243,
            "end": 247
          }
        },
        "loc": {
          "start": 243,
          "end": 247
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 250,
                "end": 258
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 265,
                      "end": 277
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 288,
                            "end": 290
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 288,
                          "end": 290
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 299,
                            "end": 307
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 299,
                          "end": 307
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 316,
                            "end": 327
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 316,
                          "end": 327
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 336,
                            "end": 340
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 336,
                          "end": 340
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 349,
                            "end": 353
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 349,
                          "end": 353
                        }
                      }
                    ],
                    "loc": {
                      "start": 278,
                      "end": 359
                    }
                  },
                  "loc": {
                    "start": 265,
                    "end": 359
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 364,
                      "end": 366
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 364,
                    "end": 366
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 371,
                      "end": 381
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 371,
                    "end": 381
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 386,
                      "end": 396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 386,
                    "end": 396
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 401,
                      "end": 409
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 401,
                    "end": 409
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 414,
                      "end": 423
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 414,
                    "end": 423
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 428,
                      "end": 440
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 428,
                    "end": 440
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 445,
                      "end": 457
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 445,
                    "end": 457
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 462,
                      "end": 474
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 462,
                    "end": 474
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 479,
                      "end": 482
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 493,
                            "end": 503
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 493,
                          "end": 503
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 512,
                            "end": 519
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 512,
                          "end": 519
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 528,
                            "end": 537
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 528,
                          "end": 537
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 546,
                            "end": 555
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 546,
                          "end": 555
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 564,
                            "end": 573
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 564,
                          "end": 573
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 582,
                            "end": 588
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 582,
                          "end": 588
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 597,
                            "end": 604
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 597,
                          "end": 604
                        }
                      }
                    ],
                    "loc": {
                      "start": 483,
                      "end": 610
                    }
                  },
                  "loc": {
                    "start": 479,
                    "end": 610
                  }
                }
              ],
              "loc": {
                "start": 259,
                "end": 612
              }
            },
            "loc": {
              "start": 250,
              "end": 612
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 613,
                "end": 615
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 613,
              "end": 615
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 616,
                "end": 626
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 616,
              "end": 626
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 627,
                "end": 637
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 627,
              "end": 637
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 638,
                "end": 647
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 638,
              "end": 647
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 648,
                "end": 659
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 648,
              "end": 659
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 660,
                "end": 666
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 676,
                      "end": 686
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 673,
                    "end": 686
                  }
                }
              ],
              "loc": {
                "start": 667,
                "end": 688
              }
            },
            "loc": {
              "start": 660,
              "end": 688
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 689,
                "end": 694
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 708,
                        "end": 720
                      }
                    },
                    "loc": {
                      "start": 708,
                      "end": 720
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 734,
                            "end": 750
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 731,
                          "end": 750
                        }
                      }
                    ],
                    "loc": {
                      "start": 721,
                      "end": 756
                    }
                  },
                  "loc": {
                    "start": 701,
                    "end": 756
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 768,
                        "end": 772
                      }
                    },
                    "loc": {
                      "start": 768,
                      "end": 772
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 786,
                            "end": 794
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 783,
                          "end": 794
                        }
                      }
                    ],
                    "loc": {
                      "start": 773,
                      "end": 800
                    }
                  },
                  "loc": {
                    "start": 761,
                    "end": 800
                  }
                }
              ],
              "loc": {
                "start": 695,
                "end": 802
              }
            },
            "loc": {
              "start": 689,
              "end": 802
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 803,
                "end": 814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 803,
              "end": 814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 815,
                "end": 829
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 815,
              "end": 829
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 830,
                "end": 835
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 830,
              "end": 835
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 836,
                "end": 845
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 836,
              "end": 845
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 846,
                "end": 850
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 860,
                      "end": 868
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 857,
                    "end": 868
                  }
                }
              ],
              "loc": {
                "start": 851,
                "end": 870
              }
            },
            "loc": {
              "start": 846,
              "end": 870
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 871,
                "end": 885
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 871,
              "end": 885
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 886,
                "end": 891
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 886,
              "end": 891
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 892,
                "end": 895
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 902,
                      "end": 911
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 902,
                    "end": 911
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 916,
                      "end": 927
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 916,
                    "end": 927
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 932,
                      "end": 943
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 932,
                    "end": 943
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 948,
                      "end": 957
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 948,
                    "end": 957
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 962,
                      "end": 969
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 962,
                    "end": 969
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 974,
                      "end": 982
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 974,
                    "end": 982
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 987,
                      "end": 999
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 987,
                    "end": 999
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1004,
                      "end": 1012
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1004,
                    "end": 1012
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1017,
                      "end": 1025
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1017,
                    "end": 1025
                  }
                }
              ],
              "loc": {
                "start": 896,
                "end": 1027
              }
            },
            "loc": {
              "start": 892,
              "end": 1027
            }
          }
        ],
        "loc": {
          "start": 248,
          "end": 1029
        }
      },
      "loc": {
        "start": 221,
        "end": 1029
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 1039,
          "end": 1055
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 1059,
            "end": 1071
          }
        },
        "loc": {
          "start": 1059,
          "end": 1071
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1074,
                "end": 1076
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1074,
              "end": 1076
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1077,
                "end": 1083
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1077,
              "end": 1083
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1084,
                "end": 1087
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 1094,
                      "end": 1107
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1094,
                    "end": 1107
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1112,
                      "end": 1121
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1112,
                    "end": 1121
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1126,
                      "end": 1137
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1126,
                    "end": 1137
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1142,
                      "end": 1151
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1142,
                    "end": 1151
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1156,
                      "end": 1165
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1156,
                    "end": 1165
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1170,
                      "end": 1177
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1170,
                    "end": 1177
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1182,
                      "end": 1194
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1182,
                    "end": 1194
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1199,
                      "end": 1207
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1199,
                    "end": 1207
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 1212,
                      "end": 1226
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1237,
                            "end": 1239
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1237,
                          "end": 1239
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1248,
                            "end": 1258
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1248,
                          "end": 1258
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 1267,
                            "end": 1277
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1267,
                          "end": 1277
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 1286,
                            "end": 1293
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1286,
                          "end": 1293
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 1302,
                            "end": 1313
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1302,
                          "end": 1313
                        }
                      }
                    ],
                    "loc": {
                      "start": 1227,
                      "end": 1319
                    }
                  },
                  "loc": {
                    "start": 1212,
                    "end": 1319
                  }
                }
              ],
              "loc": {
                "start": 1088,
                "end": 1321
              }
            },
            "loc": {
              "start": 1084,
              "end": 1321
            }
          }
        ],
        "loc": {
          "start": 1072,
          "end": 1323
        }
      },
      "loc": {
        "start": 1030,
        "end": 1323
      }
    },
    "Reminder_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Reminder_full",
        "loc": {
          "start": 1333,
          "end": 1346
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Reminder",
          "loc": {
            "start": 1350,
            "end": 1358
          }
        },
        "loc": {
          "start": 1350,
          "end": 1358
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1361,
                "end": 1363
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1361,
              "end": 1363
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1364,
                "end": 1374
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1364,
              "end": 1374
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1375,
                "end": 1385
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1375,
              "end": 1385
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1386,
                "end": 1390
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1386,
              "end": 1390
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1391,
                "end": 1402
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1391,
              "end": 1402
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 1403,
                "end": 1410
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1403,
              "end": 1410
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1411,
                "end": 1416
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1411,
              "end": 1416
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1417,
                "end": 1427
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1417,
              "end": 1427
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderItems",
              "loc": {
                "start": 1428,
                "end": 1441
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1448,
                      "end": 1450
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1448,
                    "end": 1450
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1455,
                      "end": 1465
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1455,
                    "end": 1465
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1470,
                      "end": 1480
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1470,
                    "end": 1480
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1485,
                      "end": 1489
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1485,
                    "end": 1489
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1494,
                      "end": 1505
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1494,
                    "end": 1505
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dueDate",
                    "loc": {
                      "start": 1510,
                      "end": 1517
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1510,
                    "end": 1517
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "index",
                    "loc": {
                      "start": 1522,
                      "end": 1527
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1522,
                    "end": 1527
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1532,
                      "end": 1542
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1532,
                    "end": 1542
                  }
                }
              ],
              "loc": {
                "start": 1442,
                "end": 1544
              }
            },
            "loc": {
              "start": 1428,
              "end": 1544
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 1545,
                "end": 1557
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1564,
                      "end": 1566
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1564,
                    "end": 1566
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1571,
                      "end": 1581
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1571,
                    "end": 1581
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1586,
                      "end": 1596
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1586,
                    "end": 1596
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusMode",
                    "loc": {
                      "start": 1601,
                      "end": 1610
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 1621,
                            "end": 1627
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 1642,
                                  "end": 1644
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1642,
                                "end": 1644
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 1657,
                                  "end": 1662
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1657,
                                "end": 1662
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 1675,
                                  "end": 1680
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1675,
                                "end": 1680
                              }
                            }
                          ],
                          "loc": {
                            "start": 1628,
                            "end": 1690
                          }
                        },
                        "loc": {
                          "start": 1621,
                          "end": 1690
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 1699,
                            "end": 1707
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 1725,
                                  "end": 1740
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 1722,
                                "end": 1740
                              }
                            }
                          ],
                          "loc": {
                            "start": 1708,
                            "end": 1750
                          }
                        },
                        "loc": {
                          "start": 1699,
                          "end": 1750
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1759,
                            "end": 1761
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1759,
                          "end": 1761
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1770,
                            "end": 1774
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1770,
                          "end": 1774
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1783,
                            "end": 1794
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1783,
                          "end": 1794
                        }
                      }
                    ],
                    "loc": {
                      "start": 1611,
                      "end": 1800
                    }
                  },
                  "loc": {
                    "start": 1601,
                    "end": 1800
                  }
                }
              ],
              "loc": {
                "start": 1558,
                "end": 1802
              }
            },
            "loc": {
              "start": 1545,
              "end": 1802
            }
          }
        ],
        "loc": {
          "start": 1359,
          "end": 1804
        }
      },
      "loc": {
        "start": 1324,
        "end": 1804
      }
    },
    "Resource_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Resource_list",
        "loc": {
          "start": 1814,
          "end": 1827
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Resource",
          "loc": {
            "start": 1831,
            "end": 1839
          }
        },
        "loc": {
          "start": 1831,
          "end": 1839
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1842,
                "end": 1844
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1842,
              "end": 1844
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1845,
                "end": 1850
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1845,
              "end": 1850
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "link",
              "loc": {
                "start": 1851,
                "end": 1855
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1851,
              "end": 1855
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "usedFor",
              "loc": {
                "start": 1856,
                "end": 1863
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1856,
              "end": 1863
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 1864,
                "end": 1876
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1883,
                      "end": 1885
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1883,
                    "end": 1885
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1890,
                      "end": 1898
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1890,
                    "end": 1898
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1903,
                      "end": 1914
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1903,
                    "end": 1914
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1919,
                      "end": 1923
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1919,
                    "end": 1923
                  }
                }
              ],
              "loc": {
                "start": 1877,
                "end": 1925
              }
            },
            "loc": {
              "start": 1864,
              "end": 1925
            }
          }
        ],
        "loc": {
          "start": 1840,
          "end": 1927
        }
      },
      "loc": {
        "start": 1805,
        "end": 1927
      }
    },
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 1937,
          "end": 1952
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 1956,
            "end": 1964
          }
        },
        "loc": {
          "start": 1956,
          "end": 1964
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1967,
                "end": 1969
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1967,
              "end": 1969
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1970,
                "end": 1980
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1970,
              "end": 1980
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1981,
                "end": 1991
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1981,
              "end": 1991
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 1992,
                "end": 2001
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1992,
              "end": 2001
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 2002,
                "end": 2009
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2002,
              "end": 2009
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 2010,
                "end": 2018
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2010,
              "end": 2018
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 2019,
                "end": 2029
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2036,
                      "end": 2038
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2036,
                    "end": 2038
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 2043,
                      "end": 2060
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2043,
                    "end": 2060
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 2065,
                      "end": 2077
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2065,
                    "end": 2077
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 2082,
                      "end": 2092
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2082,
                    "end": 2092
                  }
                }
              ],
              "loc": {
                "start": 2030,
                "end": 2094
              }
            },
            "loc": {
              "start": 2019,
              "end": 2094
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 2095,
                "end": 2106
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2113,
                      "end": 2115
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2113,
                    "end": 2115
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 2120,
                      "end": 2134
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2120,
                    "end": 2134
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 2139,
                      "end": 2147
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2139,
                    "end": 2147
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 2152,
                      "end": 2161
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2152,
                    "end": 2161
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 2166,
                      "end": 2176
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2166,
                    "end": 2176
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 2181,
                      "end": 2186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2181,
                    "end": 2186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 2191,
                      "end": 2198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2191,
                    "end": 2198
                  }
                }
              ],
              "loc": {
                "start": 2107,
                "end": 2200
              }
            },
            "loc": {
              "start": 2095,
              "end": 2200
            }
          }
        ],
        "loc": {
          "start": 1965,
          "end": 2202
        }
      },
      "loc": {
        "start": 1928,
        "end": 2202
      }
    },
    "Schedule_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_list",
        "loc": {
          "start": 2212,
          "end": 2225
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 2229,
            "end": 2237
          }
        },
        "loc": {
          "start": 2229,
          "end": 2237
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2240,
                "end": 2246
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 2256,
                      "end": 2266
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2253,
                    "end": 2266
                  }
                }
              ],
              "loc": {
                "start": 2247,
                "end": 2268
              }
            },
            "loc": {
              "start": 2240,
              "end": 2268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2269,
                "end": 2271
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2269,
              "end": 2271
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2272,
                "end": 2282
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2272,
              "end": 2282
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2283,
                "end": 2293
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2283,
              "end": 2293
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 2294,
                "end": 2303
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2294,
              "end": 2303
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 2304,
                "end": 2311
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2304,
              "end": 2311
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 2312,
                "end": 2320
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2312,
              "end": 2320
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 2321,
                "end": 2331
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2338,
                      "end": 2340
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2338,
                    "end": 2340
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 2345,
                      "end": 2362
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2345,
                    "end": 2362
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 2367,
                      "end": 2379
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2367,
                    "end": 2379
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 2384,
                      "end": 2394
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2384,
                    "end": 2394
                  }
                }
              ],
              "loc": {
                "start": 2332,
                "end": 2396
              }
            },
            "loc": {
              "start": 2321,
              "end": 2396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 2397,
                "end": 2408
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2415,
                      "end": 2417
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2415,
                    "end": 2417
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 2422,
                      "end": 2436
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2422,
                    "end": 2436
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 2441,
                      "end": 2449
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2441,
                    "end": 2449
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 2454,
                      "end": 2463
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2454,
                    "end": 2463
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 2468,
                      "end": 2478
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2468,
                    "end": 2478
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 2483,
                      "end": 2488
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2483,
                    "end": 2488
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 2493,
                      "end": 2500
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2493,
                    "end": 2500
                  }
                }
              ],
              "loc": {
                "start": 2409,
                "end": 2502
              }
            },
            "loc": {
              "start": 2397,
              "end": 2502
            }
          }
        ],
        "loc": {
          "start": 2238,
          "end": 2504
        }
      },
      "loc": {
        "start": 2203,
        "end": 2504
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 2514,
          "end": 2522
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 2526,
            "end": 2529
          }
        },
        "loc": {
          "start": 2526,
          "end": 2529
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2532,
                "end": 2534
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2532,
              "end": 2534
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2535,
                "end": 2545
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2535,
              "end": 2545
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 2546,
                "end": 2549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2546,
              "end": 2549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2550,
                "end": 2559
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2550,
              "end": 2559
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2560,
                "end": 2572
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2579,
                      "end": 2581
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2579,
                    "end": 2581
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2586,
                      "end": 2594
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2586,
                    "end": 2594
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2599,
                      "end": 2610
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2599,
                    "end": 2610
                  }
                }
              ],
              "loc": {
                "start": 2573,
                "end": 2612
              }
            },
            "loc": {
              "start": 2560,
              "end": 2612
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2613,
                "end": 2616
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 2623,
                      "end": 2628
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2623,
                    "end": 2628
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2633,
                      "end": 2645
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2633,
                    "end": 2645
                  }
                }
              ],
              "loc": {
                "start": 2617,
                "end": 2647
              }
            },
            "loc": {
              "start": 2613,
              "end": 2647
            }
          }
        ],
        "loc": {
          "start": 2530,
          "end": 2649
        }
      },
      "loc": {
        "start": 2505,
        "end": 2649
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 2659,
          "end": 2667
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 2671,
            "end": 2675
          }
        },
        "loc": {
          "start": 2671,
          "end": 2675
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2678,
                "end": 2680
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2678,
              "end": 2680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 2681,
                "end": 2686
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2681,
              "end": 2686
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2687,
                "end": 2691
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2687,
              "end": 2691
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2692,
                "end": 2698
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2692,
              "end": 2698
            }
          }
        ],
        "loc": {
          "start": 2676,
          "end": 2700
        }
      },
      "loc": {
        "start": 2650,
        "end": 2700
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "home",
      "loc": {
        "start": 2708,
        "end": 2712
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 2714,
              "end": 2719
            }
          },
          "loc": {
            "start": 2713,
            "end": 2719
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "HomeInput",
              "loc": {
                "start": 2721,
                "end": 2730
              }
            },
            "loc": {
              "start": 2721,
              "end": 2730
            }
          },
          "loc": {
            "start": 2721,
            "end": 2731
          }
        },
        "directives": [],
        "loc": {
          "start": 2713,
          "end": 2731
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "home",
            "loc": {
              "start": 2737,
              "end": 2741
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 2742,
                  "end": 2747
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 2750,
                    "end": 2755
                  }
                },
                "loc": {
                  "start": 2749,
                  "end": 2755
                }
              },
              "loc": {
                "start": 2742,
                "end": 2755
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "notes",
                  "loc": {
                    "start": 2763,
                    "end": 2768
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Note_list",
                        "loc": {
                          "start": 2782,
                          "end": 2791
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 2779,
                        "end": 2791
                      }
                    }
                  ],
                  "loc": {
                    "start": 2769,
                    "end": 2797
                  }
                },
                "loc": {
                  "start": 2763,
                  "end": 2797
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "reminders",
                  "loc": {
                    "start": 2802,
                    "end": 2811
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Reminder_full",
                        "loc": {
                          "start": 2825,
                          "end": 2838
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 2822,
                        "end": 2838
                      }
                    }
                  ],
                  "loc": {
                    "start": 2812,
                    "end": 2844
                  }
                },
                "loc": {
                  "start": 2802,
                  "end": 2844
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "resources",
                  "loc": {
                    "start": 2849,
                    "end": 2858
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Resource_list",
                        "loc": {
                          "start": 2872,
                          "end": 2885
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 2869,
                        "end": 2885
                      }
                    }
                  ],
                  "loc": {
                    "start": 2859,
                    "end": 2891
                  }
                },
                "loc": {
                  "start": 2849,
                  "end": 2891
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "schedules",
                  "loc": {
                    "start": 2896,
                    "end": 2905
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Schedule_list",
                        "loc": {
                          "start": 2919,
                          "end": 2932
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 2916,
                        "end": 2932
                      }
                    }
                  ],
                  "loc": {
                    "start": 2906,
                    "end": 2938
                  }
                },
                "loc": {
                  "start": 2896,
                  "end": 2938
                }
              }
            ],
            "loc": {
              "start": 2757,
              "end": 2942
            }
          },
          "loc": {
            "start": 2737,
            "end": 2942
          }
        }
      ],
      "loc": {
        "start": 2733,
        "end": 2944
      }
    },
    "loc": {
      "start": 2702,
      "end": 2944
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_home"
  }
} as const;
